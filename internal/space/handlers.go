package space

import (
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"yoomates/internal/auth"
)

// ViewEngine handles rendering of My Space templates and partials
type ViewEngine struct {
	templates *template.Template
	repo      *Repository
}

// Simple, fast Markdown-to-HTML renderer for document reading/preview
func renderMarkdown(input string) template.HTML {
	if strings.TrimSpace(input) == "" {
		return template.HTML("<p class=\"text-muted\">Empty document. Start typing...</p>")
	}

	escaped := html.EscapeString(input)
	lines := strings.Split(escaped, "\n")
	var out strings.Builder
	inList := false
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Code block toggle
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				out.WriteString("</code></pre>\n")
				inCodeBlock = false
			} else {
				if inList {
					out.WriteString("</ul>\n")
					inList = false
				}
				out.WriteString("<pre class=\"md-code-block\"><code>")
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			out.WriteString(line + "\n")
			continue
		}

		// Headings
		if strings.HasPrefix(trimmed, "### ") {
			if inList {
				out.WriteString("</ul>\n")
				inList = false
			}
			out.WriteString(fmt.Sprintf("<h3 class=\"md-h3\">%s</h3>\n", trimmed[4:]))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			if inList {
				out.WriteString("</ul>\n")
				inList = false
			}
			out.WriteString(fmt.Sprintf("<h2 class=\"md-h2\">%s</h2>\n", trimmed[3:]))
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			if inList {
				out.WriteString("</ul>\n")
				inList = false
			}
			out.WriteString(fmt.Sprintf("<h1 class=\"md-h1\">%s</h1>\n", trimmed[2:]))
			continue
		}

		// Task lists
		if strings.HasPrefix(trimmed, "- [x] ") {
			if !inList {
				out.WriteString("<ul class=\"md-task-list\">\n")
				inList = true
			}
			out.WriteString(fmt.Sprintf("<li class=\"task-item checked\"><span class=\"task-box\">☑</span> %s</li>\n", trimmed[6:]))
			continue
		}
		if strings.HasPrefix(trimmed, "- [ ] ") {
			if !inList {
				out.WriteString("<ul class=\"md-task-list\">\n")
				inList = true
			}
			out.WriteString(fmt.Sprintf("<li class=\"task-item\"><span class=\"task-box\">☐</span> %s</li>\n", trimmed[6:]))
			continue
		}

		// Standard unordered list
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if !inList {
				out.WriteString("<ul class=\"md-list\">\n")
				inList = true
			}
			out.WriteString(fmt.Sprintf("<li>%s</li>\n", trimmed[2:]))
			continue
		}

		if inList && trimmed == "" {
			out.WriteString("</ul>\n")
			inList = false
			continue
		}

		// Empty lines
		if trimmed == "" {
			out.WriteString("<div class=\"md-spacer\"></div>\n")
			continue
		}

		// Inline bold & code replacements
		formattedLine := trimmed
		reBold := regexp.MustCompile(`\*\*(.*?)\*\*`)
		formattedLine = reBold.ReplaceAllString(formattedLine, "<strong>$1</strong>")

		reCode := regexp.MustCompile("`(.*?)`")
		formattedLine = reCode.ReplaceAllString(formattedLine, "<code class=\"md-inline-code\">$1</code>")

		out.WriteString(fmt.Sprintf("<p class=\"md-p\">%s</p>\n", formattedLine))
	}

	if inList {
		out.WriteString("</ul>\n")
	}
	if inCodeBlock {
		out.WriteString("</code></pre>\n")
	}

	return template.HTML(out.String())
}

// NewViewEngine parses templates for My Space
func NewViewEngine(templatesDir string, repo *Repository) (*ViewEngine, error) {
	funcMap := template.FuncMap{
		"isImage": func(url string) bool {
			lower := strings.ToLower(url)
			return strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") ||
				strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".gif") ||
				strings.HasSuffix(lower, ".webp")
		},
		"formatPercent": func(p float64) string {
			return fmt.Sprintf("%.1f%%", p)
		},
		"formatSize": func(bytes int) string {
			if bytes <= 0 {
				return ""
			}
			if bytes < 1024 {
				return fmt.Sprintf("%d B", bytes)
			} else if bytes < 1024*1024 {
				return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
			}
			return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
		},
		"renderMarkdown": renderMarkdown,
	}

	basePattern := filepath.Join(templatesDir, "base.html")
	pagePattern := filepath.Join(templatesDir, "myspace.html")
	partialsPattern := filepath.Join(templatesDir, "partials", "*.html")

	tmpl := template.New("").Funcs(funcMap)
	tmpl, err := tmpl.ParseFiles(basePattern, pagePattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base and myspace templates: %w", err)
	}

	tmpl, err = tmpl.ParseGlob(partialsPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse partial templates: %w", err)
	}

	return &ViewEngine{
		templates: tmpl,
		repo:      repo,
	}, nil
}

func getCurrentUser(r *http.Request) CurrentUserItem {
	userCtx, _ := auth.GetUserFromContext(r.Context())
	if userCtx != nil {
		initials := "U"
		if len(userCtx.Name) > 0 {
			initials = strings.ToUpper(string(userCtx.Name[0]))
		} else if len(userCtx.Username) > 0 {
			initials = strings.ToUpper(string(userCtx.Username[0]))
		}
		return CurrentUserItem{
			ID:       userCtx.ID,
			Name:     userCtx.Name,
			Username: userCtx.Username,
			Initials: initials,
		}
	}

	return CurrentUserItem{
		ID:       "usr_guest",
		Name:     "Team Member",
		Username: "member",
		Initials: "TM",
	}
}

// MySpaceHandler renders the master My Space workspace interface
func (v *ViewEngine) MySpaceHandler(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)

	folders, err := v.repo.GetFolders(user.ID)
	if err != nil {
		log.Printf("[space-error] Failed to fetch folders: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	folderID := r.URL.Query().Get("folder_id")
	docID := r.URL.Query().Get("doc_id")
	viewType := r.URL.Query().Get("view")

	var currentFolder *DocumentFolder
	if folderID != "" {
		for _, f := range folders {
			if f.ID == folderID {
				currentFolder = f
				break
			}
		}
	}
	if currentFolder == nil && len(folders) > 0 {
		currentFolder = folders[0]
		folderID = currentFolder.ID
	}

	docs, err := v.repo.GetDocuments(user.ID, folderID)
	if err != nil {
		log.Printf("[space-error] Failed to fetch documents: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var activeDoc *Document
	if docID != "" {
		activeDoc, _ = v.repo.GetDocument(docID, user.ID)
	}
	if activeDoc == nil && len(docs) > 0 {
		activeDoc = docs[0]
	}

	if viewType == "" {
		if docID != "" || activeDoc != nil {
			viewType = "editor"
		} else {
			viewType = "kanban"
		}
	}

	model := SpaceViewModel{
		CurrentUser:    user,
		Folders:        folders,
		CurrentFolder:  currentFolder,
		ActiveDocument: activeDoc,
		ViewType:       viewType,
		Documents:      docs,
		KanbanColumns:  v.repo.BuildKanbanColumns(docs),
		TotalDocs:      len(docs),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := v.templates.ExecuteTemplate(w, "myspace.html", model); err != nil {
		log.Printf("[space-error] Failed to render myspace.html: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// GetDocumentHandler loads a document into the editor pane via HTMX
func (v *ViewEngine) GetDocumentHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid document id", http.StatusBadRequest)
		return
	}
	docID := parts[3]
	user := getCurrentUser(r)

	doc, err := v.repo.GetDocument(docID, user.ID)
	if err != nil {
		http.Error(w, "document not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := v.templates.ExecuteTemplate(w, "space_editor", doc); err != nil {
		log.Printf("[space-error] Failed to render space_editor: %v\n", err)
	}
}

// CreateDocumentHandler creates a new document
func (v *ViewEngine) CreateDocumentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user := getCurrentUser(r)
	title := r.FormValue("title")
	folderID := r.FormValue("folder_id")
	status := r.FormValue("status")

	doc, err := v.repo.CreateDocument(user.ID, folderID, title, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If HTMX requested, return the new editor view directly
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = v.templates.ExecuteTemplate(w, "space_editor", doc)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/myspace?doc_id=%s&folder_id=%s", doc.ID, folderID), http.StatusSeeOther)
}

// AutosaveHandler updates document content and title with a 0ms response
func (v *ViewEngine) AutosaveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid document id", http.StatusBadRequest)
		return
	}
	docID := parts[3]
	user := getCurrentUser(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")
	status := r.FormValue("status")
	icon := r.FormValue("icon")

	doc, err := v.repo.SaveDocument(docID, user.ID, title, content, icon, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Return subtle saved status pill and update title in sidebar if changed
	responseHTML := fmt.Sprintf(`
<span id="autosave-status" class="autosave-indicator saved">
  <span class="status-dot"></span> Saved %s
</span>
<span id="doc-title-badge-%s" hx-swap-oob="innerHTML">%s</span>
`, doc.FormattedTime, doc.ID, html.EscapeString(doc.Title))

	_, _ = w.Write([]byte(responseHTML))
}

// StatusHandler updates document status (Kanban move)
func (v *ViewEngine) StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid document id", http.StatusBadRequest)
		return
	}
	docID := parts[3]
	user := getCurrentUser(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	status := r.FormValue("status")

	doc, err := v.repo.UpdateDocumentStatus(docID, user.ID, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// If requested from Kanban view, re-render the Kanban board
	folderID := r.FormValue("folder_id")
	if folderID == "" {
		folderID = doc.FolderID
	}
	docs, _ := v.repo.GetDocuments(user.ID, folderID)
	kanbanCols := v.repo.BuildKanbanColumns(docs)

	if err := v.templates.ExecuteTemplate(w, "space_kanban", kanbanCols); err != nil {
		log.Printf("[space-error] Failed to render space_kanban: %v\n", err)
	}
}

// KanbanViewHandler renders the Kanban board partial
func (v *ViewEngine) KanbanViewHandler(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	folderID := r.URL.Query().Get("folder_id")

	docs, err := v.repo.GetDocuments(user.ID, folderID)
	if err != nil {
		http.Error(w, "failed to get documents", http.StatusInternalServerError)
		return
	}

	kanbanCols := v.repo.BuildKanbanColumns(docs)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := v.templates.ExecuteTemplate(w, "space_kanban", kanbanCols); err != nil {
		log.Printf("[space-error] Failed to render space_kanban: %v\n", err)
	}
}

// ListViewHandler renders the List/Table partial
func (v *ViewEngine) ListViewHandler(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	folderID := r.URL.Query().Get("folder_id")

	docs, err := v.repo.GetDocuments(user.ID, folderID)
	if err != nil {
		http.Error(w, "failed to get documents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := v.templates.ExecuteTemplate(w, "space_list", docs); err != nil {
		log.Printf("[space-error] Failed to render space_list: %v\n", err)
	}
}
