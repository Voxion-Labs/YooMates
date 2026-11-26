package feed

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"yoomates/internal/auth"
)

// ViewEngine handles rendering of Global Feed templates and HTMX partials
type ViewEngine struct {
	templates *template.Template
	repo      *Repository
}

// NewViewEngine parses all templates for the Global Feed
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
		"add": func(a, b int) int {
			return a + b
		},
		"renderMarkdown": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	basePattern := filepath.Join(templatesDir, "base.html")
	pagePattern := filepath.Join(templatesDir, "global.html")
	partialsPattern := filepath.Join(templatesDir, "partials", "*.html")

	tmpl := template.New("").Funcs(funcMap)
	tmpl, err := tmpl.ParseFiles(basePattern, pagePattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base and global templates: %w", err)
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

// GetCurrentUser retrieves authenticated user or returns guest fallback
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

// GlobalHandler renders the master Twitter/X style Global Feed surface
func (v *ViewEngine) GlobalHandler(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)

	posts, nextCursor, err := v.repo.GetPosts("", 10, user.ID)
	if err != nil {
		log.Printf("[feed-error] Failed to fetch initial posts: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	model := FeedViewModel{
		CurrentUser: user,
		Posts:       posts,
		NextCursor:  nextCursor,
		HasMore:     nextCursor != "",
		TrendingTopics: []TrendingItem{
			{Category: "Architecture · Trending", Topic: "#ZeroNodeModules", PostCount: "14.2K posts"},
			{Category: "Technology · Trending", Topic: "#GolangHTMX", PostCount: "28.5K posts"},
			{Category: "Performance · Trending", Topic: "#WebAssembly", PostCount: "9.1K posts"},
			{Category: "Audio · Trending", Topic: "#CplusplusWebRTC", PostCount: "5.8K posts"},
		},
		WhoToFollow: []FollowSuggestion{
			{ID: "usr_voxion", Name: "Voxion Bot", Username: "voxion-bot", Initials: "VB", Bio: "Automated engine coordinator & CI manifest engine"},
			{ID: "usr_rjena", Name: "Rudranarayan Jena", Username: "rjena", Initials: "RJ", Bio: "Systems architect & builder @ Voxion Labs"},
			{ID: "usr_liam", Name: "Liam Brooks", Username: "liambrooks", Initials: "LB", Bio: "Distributed backend & media systems engineer"},
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := v.templates.ExecuteTemplate(w, "global.html", model); err != nil {
		log.Printf("[feed-error] Failed to render global.html: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// PostsStreamHandler handles infinite scroll pagination via HTMX
func (v *ViewEngine) PostsStreamHandler(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	cursor := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	posts, nextCursor, err := v.repo.GetPosts(cursor, limit, user.ID)
	if err != nil {
		log.Printf("[feed-error] Failed to paginate posts: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	for _, p := range posts {
		if err := v.templates.ExecuteTemplate(w, "feed_post", p); err != nil {
			log.Printf("[feed-error] Failed to render feed_post: %v\n", err)
			return
		}
	}

	// Render next infinite scroll sentinel trigger if more posts exist
	if nextCursor != "" {
		sentinel := fmt.Sprintf(`
<div id="feed-scroll-sentinel" 
     class="feed-scroll-sentinel" 
     hx-get="/global/posts?cursor=%s" 
     hx-trigger="revealed" 
     hx-swap="outerHTML">
  <span class="feed-spinner"></span>
</div>
`, nextCursor)
		_, _ = w.Write([]byte(sentinel))
	}
}

// CreatePostHandler creates a new post and returns rendered HTML
func (v *ViewEngine) CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user := getCurrentUser(r)
	content := r.FormValue("content")
	mediaURL := r.FormValue("media_url")
	mediaType := r.FormValue("media_type")
	mediaName := r.FormValue("media_name")
	quotePostID := r.FormValue("quote_post_id")

	var pollInput *CreatePollInput
	pollQuestion := r.FormValue("poll_question")
	if pollQuestion != "" {
		var options []string
		for i := 0; i < 4; i++ {
			optVal := r.FormValue(fmt.Sprintf("poll_option_%d", i))
			if strings.TrimSpace(optVal) != "" {
				options = append(options, optVal)
			}
		}
		if len(options) >= 2 {
			pollInput = &CreatePollInput{
				Question: pollQuestion,
				Options:  options,
			}
		}
	}

	post, err := v.repo.CreatePost(user.ID, user.Name, user.Username, content, mediaURL, mediaType, mediaName, quotePostID, pollInput)
	if err != nil {
		log.Printf("[feed-error] Failed to create post: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// If requested via HTMX, return the newly created post partial for prepend
	if err := v.templates.ExecuteTemplate(w, "feed_post", post); err != nil {
		log.Printf("[feed-error] Failed to render new feed_post: %v\n", err)
	}
}

// LikeHandler toggles like state and renders updated like button partial
func (v *ViewEngine) LikeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}
	postID := parts[3]

	user := getCurrentUser(r)
	hasLiked, count, err := v.repo.ToggleLike(postID, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	activeClass := ""
	heartIcon := "🤍"
	if hasLiked {
		activeClass = "liked"
		heartIcon = "❤️"
	}

	html := fmt.Sprintf(`
<button id="like-btn-%s" 
        class="post-action-btn like-btn %s" 
        hx-post="/global/posts/%s/like" 
        hx-target="#like-btn-%s" 
        hx-swap="outerHTML" 
        title="Like post">
  <span class="action-icon">%s</span>
  <span class="action-count">%d</span>
</button>
`, postID, activeClass, postID, postID, heartIcon, count)

	_, _ = w.Write([]byte(html))
}

// CommentsHandler loads or adds comments for a post
func (v *ViewEngine) CommentsHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}
	postID := parts[3]

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		user := getCurrentUser(r)
		text := r.FormValue("text")
		cmt, err := v.repo.AddComment(postID, user.ID, user.Name, user.Username, text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := v.templates.ExecuteTemplate(w, "comment_item", cmt); err != nil {
			log.Printf("[feed-error] Failed to render comment_item: %v\n", err)
		}
		return
	}

	// GET comments
	comments, err := v.repo.GetComments(postID)
	if err != nil {
		http.Error(w, "failed to get comments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		PostID   string
		Comments []*Comment
	}{
		PostID:   postID,
		Comments: comments,
	}

	if err := v.templates.ExecuteTemplate(w, "feed_comments", data); err != nil {
		log.Printf("[feed-error] Failed to render feed_comments: %v\n", err)
	}
}

// PollVoteHandler records a vote and returns updated poll component
func (v *ViewEngine) PollVoteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid poll id", http.StatusBadRequest)
		return
	}
	pollID := parts[3]
	optionID := r.FormValue("option_id")

	user := getCurrentUser(r)
	poll, err := v.repo.VotePoll(pollID, optionID, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := v.templates.ExecuteTemplate(w, "feed_poll", poll); err != nil {
		log.Printf("[feed-error] Failed to render feed_poll: %v\n", err)
	}
}
