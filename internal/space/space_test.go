package space

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpaceRepository_InitialSeed(t *testing.T) {
	repo := NewRepository(nil)

	folders, err := repo.GetFolders("usr_test")
	if err != nil {
		t.Fatalf("unexpected error fetching folders: %v", err)
	}
	if len(folders) < 3 {
		t.Fatalf("expected at least 3 default folders, got %d", len(folders))
	}

	docs, err := repo.GetDocuments("usr_test", "")
	if err != nil {
		t.Fatalf("unexpected error fetching documents: %v", err)
	}
	if len(docs) < 4 {
		t.Fatalf("expected at least 4 default documents, got %d", len(docs))
	}
}

func TestSpaceRepository_CreateAndGet(t *testing.T) {
	repo := NewRepository(nil)

	doc, err := repo.CreateDocument("usr_test", "fld_specs", "Sprint Roadmap v2", "TODO")
	if err != nil {
		t.Fatalf("unexpected error creating document: %v", err)
	}

	if doc.ID == "" {
		t.Errorf("expected document to have ID")
	}
	if doc.Title != "Sprint Roadmap v2" {
		t.Errorf("expected title 'Sprint Roadmap v2', got %s", doc.Title)
	}
	if doc.Status != "TODO" {
		t.Errorf("expected status 'TODO', got %s", doc.Status)
	}

	fetched, err := repo.GetDocument(doc.ID, "usr_test")
	if err != nil {
		t.Fatalf("unexpected error fetching created doc: %v", err)
	}
	if fetched.Title != doc.Title {
		t.Errorf("expected title match: %s vs %s", fetched.Title, doc.Title)
	}
}

func TestSpaceRepository_Autosave(t *testing.T) {
	repo := NewRepository(nil)

	saved, err := repo.SaveDocument("doc_arch_spec", "usr_test", "Updated Architecture Spec", "# New Content\n\nAutosaved markdown", "📑", "IN_PROGRESS")
	if err != nil {
		t.Fatalf("unexpected error autosaving: %v", err)
	}

	if saved.Title != "Updated Architecture Spec" {
		t.Errorf("expected updated title, got %s", saved.Title)
	}
	if !strings.Contains(saved.Content, "Autosaved markdown") {
		t.Errorf("expected content to contain 'Autosaved markdown'")
	}
}

func TestSpaceRepository_StatusChange(t *testing.T) {
	repo := NewRepository(nil)

	updated, err := repo.UpdateDocumentStatus("doc_sprint_tasks", "usr_test", "DONE")
	if err != nil {
		t.Fatalf("unexpected error updating status: %v", err)
	}
	if updated.Status != "DONE" {
		t.Errorf("expected status 'DONE', got %s", updated.Status)
	}
}

func TestSpaceRepository_BuildKanbanColumns(t *testing.T) {
	repo := NewRepository(nil)

	docs, _ := repo.GetDocuments("usr_test", "")
	cols := repo.BuildKanbanColumns(docs)

	if len(cols) != 4 {
		t.Fatalf("expected 4 Kanban columns, got %d", len(cols))
	}

	total := 0
	for _, c := range cols {
		total += c.Count
	}
	if total != len(docs) {
		t.Errorf("expected total cards %d to match doc count %d", total, len(docs))
	}
}

func TestViewEngine_RenderMySpace(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "web", "templates")
	repo := NewRepository(nil)
	engine, err := NewViewEngine(templatesDir, repo)
	if err != nil {
		t.Fatalf("failed to create ViewEngine: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/myspace", nil)
	rr := httptest.NewRecorder()

	engine.MySpaceHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "space-shell") {
		t.Error("expected body to contain space-shell class")
	}
	if !strings.Contains(body, "space-sidebar") {
		t.Error("expected body to contain space-sidebar")
	}
	if !strings.Contains(body, "workspace-view-container") {
		t.Error("expected body to contain workspace-view-container")
	}
}

func TestViewEngine_GetDocumentHandler(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "web", "templates")
	repo := NewRepository(nil)
	engine, err := NewViewEngine(templatesDir, repo)
	if err != nil {
		t.Fatalf("failed to create ViewEngine: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/myspace/documents/doc_arch_spec", nil)
	rr := httptest.NewRecorder()

	engine.GetDocumentHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "space-editor-surface") {
		t.Error("expected body to contain space-editor-surface")
	}
	if !strings.Contains(body, "doc-content-textarea") {
		t.Error("expected body to contain doc-content-textarea")
	}
}

func TestViewEngine_AutosaveHandler(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "web", "templates")
	repo := NewRepository(nil)
	engine, err := NewViewEngine(templatesDir, repo)
	if err != nil {
		t.Fatalf("failed to create ViewEngine: %v", err)
	}

	form := url.Values{}
	form.Set("title", "Autosaved Title")
	form.Set("content", "## Real-time Autosaved Content")
	req := httptest.NewRequest(http.MethodPost, "/myspace/documents/doc_arch_spec/autosave", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	engine.AutosaveHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Saved") {
		t.Error("expected body to contain 'Saved'")
	}
	if !strings.Contains(body, "Autosaved Title") {
		t.Error("expected body to contain new title")
	}
}

func TestViewEngine_KanbanAndListView(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "web", "templates")
	repo := NewRepository(nil)
	engine, err := NewViewEngine(templatesDir, repo)
	if err != nil {
		t.Fatalf("failed to create ViewEngine: %v", err)
	}

	// Kanban view
	kReq := httptest.NewRequest(http.MethodGet, "/myspace/view/kanban", nil)
	kRR := httptest.NewRecorder()
	engine.KanbanViewHandler(kRR, kReq)
	if kRR.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", kRR.Code)
	}
	if !strings.Contains(kRR.Body.String(), "space-kanban-board") {
		t.Error("expected response to contain space-kanban-board")
	}

	// List view
	lReq := httptest.NewRequest(http.MethodGet, "/myspace/view/list", nil)
	lRR := httptest.NewRecorder()
	engine.ListViewHandler(lRR, lReq)
	if lRR.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", lRR.Code)
	}
	if !strings.Contains(lRR.Body.String(), "space-list-view") {
		t.Error("expected response to contain space-list-view")
	}
}
