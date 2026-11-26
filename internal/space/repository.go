package space

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Repository manages persistence for My Space (MySQL + In-Memory Fallback)
type Repository struct {
	db        *sql.DB
	mockMu    sync.RWMutex
	folders   map[string]*DocumentFolder
	documents map[string]*Document
}

func randomID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "Just now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	return t.Format("Jan 02, 15:04")
}

// NewRepository creates a new My Space repository instance
func NewRepository(db *sql.DB) *Repository {
	repo := &Repository{
		db:        db,
		folders:   make(map[string]*DocumentFolder),
		documents: make(map[string]*Document),
	}

	repo.seedInitialData()
	return repo
}

func (r *Repository) seedInitialData() {
	now := time.Now().UTC()

	// Default Folders
	f1 := &DocumentFolder{
		ID:        "fld_specs",
		Name:      "Specifications",
		Color:     "#6B7280",
		Position:  0,
		UserID:    "usr_default",
		DocCount:  2,
		CreatedAt: now.Add(-48 * time.Hour),
		UpdatedAt: now.Add(-2 * time.Hour),
	}
	f2 := &DocumentFolder{
		ID:        "fld_sprint",
		Name:      "Sprint Planning",
		Color:     "#10B981",
		Position:  1,
		UserID:    "usr_default",
		DocCount:  2,
		CreatedAt: now.Add(-24 * time.Hour),
		UpdatedAt: now.Add(-1 * time.Hour),
	}
	f3 := &DocumentFolder{
		ID:        "fld_notes",
		Name:      "Quick Notes",
		Color:     "#3B82F6",
		Position:  2,
		UserID:    "usr_default",
		DocCount:  0,
		CreatedAt: now.Add(-12 * time.Hour),
		UpdatedAt: now.Add(-12 * time.Hour),
	}

	// Default Documents
	d1 := &Document{
		ID:       "doc_arch_spec",
		Title:    "Engine Architecture & Resilience Specification",
		FolderID: f1.ID,
		Icon:     "📄",
		Status:   "IN_PROGRESS",
		Position: 0,
		UserID:   "usr_default",
		Content: `# Engine Architecture Specification

## Overview
This document outlines the high-throughput, low-latency architecture replacing monolithic JavaScript runtimes with high-concurrency Go and C++.

### Core Principles
- **Zero Runtime Dependencies**: No runtime node_modules or transpiled JavaScript frameworks.
- **Dumb Pipe Transport**: Minimalist JavaScript strictly under 230 lines acting solely as an event relay.
- **Server-Driven Hypermedia**: Go HTML templates with HTMX for instantaneous sub-5ms DOM swaps.
- **Native Audio Pipeline**: Low-latency C++ WebRTC packet multiplexing and adaptive jitter buffers.

### Verification Checklist
- [x] Native Go session-based authentication (Argon2id)
- [x] Goroutine-backed WebSocket Hub
- [x] C++ RTP RFC 3550 audio packet processor
- [x] Opus TOC parser and adaptive jitter buffer
- [ ] Offline local-first workspace autosave
`,
		CreatedAt:     now.Add(-24 * time.Hour),
		UpdatedAt:     now.Add(-15 * time.Minute),
		FormattedTime: "15m ago",
	}

	d2 := &Document{
		ID:       "doc_audio_log",
		Title:    "Audio Buffer & Silence Suppression Benchmarks",
		FolderID: f1.ID,
		Icon:     "🔊",
		Status:   "DONE",
		Position: 1,
		UserID:   "usr_default",
		Content: `# Audio Buffer & Silence Suppression Benchmarks

## Benchmark Summary
- **Network Jitter**: RFC 3550 moving variance computed with $J(i) = J(i-1) + (|D| - J(i-1))/16$.
- **Bandwidth Reduction**: Silence suppression via DTX controller saves ~68% voice bandwidth during conversational pauses.
- **Comfort Noise Generation**: XorShift32 PRNG shaped at -55 dBFS prevents perceived call dropouts.
`,
		CreatedAt:     now.Add(-48 * time.Hour),
		UpdatedAt:     now.Add(-2 * time.Hour),
		FormattedTime: "2h ago",
	}

	d3 := &Document{
		ID:       "doc_sprint_tasks",
		Title:    "Workspace & Feed Release Tasks",
		FolderID: f2.ID,
		Icon:     "📋",
		Status:   "TODO",
		Position: 0,
		UserID:   "usr_default",
		Content: `# Workspace Release Tasks

1. Implement Kanban board status transitions
2. Wire automatic background autosave on typing
3. Verify local IndexedDB cache synchronization
`,
		CreatedAt:     now.Add(-6 * time.Hour),
		UpdatedAt:     now.Add(-1 * time.Hour),
		FormattedTime: "1h ago",
	}

	d4 := &Document{
		ID:       "doc_backlog_ideas",
		Title:    "Federated Relay & Distributed Sync",
		FolderID: f2.ID,
		Icon:     "💡",
		Status:   "BACKLOG",
		Position: 1,
		UserID:   "usr_default",
		Content: `# Future Protocol Ideas

Exploration of CRDT-based offline peer-to-peer document synchronization and cryptographic signature verification.
`,
		CreatedAt:     now.Add(-12 * time.Hour),
		UpdatedAt:     now.Add(-4 * time.Hour),
		FormattedTime: "4h ago",
	}

	r.folders[f1.ID] = f1
	r.folders[f2.ID] = f2
	r.folders[f3.ID] = f3

	r.documents[d1.ID] = d1
	r.documents[d2.ID] = d2
	r.documents[d3.ID] = d3
	r.documents[d4.ID] = d4
}

// GetFolders returns all document folders for a user
func (r *Repository) GetFolders(userID string) ([]*DocumentFolder, error) {
	r.mockMu.RLock()
	defer r.mockMu.RUnlock()

	var list []*DocumentFolder
	for _, f := range r.folders {
		// Calculate live doc count
		count := 0
		for _, d := range r.documents {
			if !d.IsArchived && d.FolderID == f.ID {
				count++
			}
		}
		fCopy := *f
		fCopy.DocCount = count
		list = append(list, &fCopy)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Position < list[j].Position
	})

	return list, nil
}

// CreateFolder creates a new folder
func (r *Repository) CreateFolder(userID, name, color string) (*DocumentFolder, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, errors.New("folder name must not be empty")
	}

	r.mockMu.Lock()
	defer r.mockMu.Unlock()

	id := randomID("fld")
	folder := &DocumentFolder{
		ID:        id,
		Name:      cleanName,
		Color:     color,
		Position:  len(r.folders),
		UserID:    userID,
		DocCount:  0,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	r.folders[id] = folder
	return folder, nil
}

// GetDocuments returns all documents in a folder (or all if folderID is empty)
func (r *Repository) GetDocuments(userID, folderID string) ([]*Document, error) {
	r.mockMu.RLock()
	defer r.mockMu.RUnlock()

	var list []*Document
	for _, d := range r.documents {
		if d.IsArchived {
			continue
		}
		if folderID != "" && d.FolderID != folderID {
			continue
		}
		dCopy := *d
		dCopy.FormattedTime = formatTimeAgo(dCopy.UpdatedAt)
		list = append(list, &dCopy)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})

	return list, nil
}

// GetDocument returns a single document by ID
func (r *Repository) GetDocument(id, userID string) (*Document, error) {
	r.mockMu.RLock()
	defer r.mockMu.RUnlock()

	doc, ok := r.documents[id]
	if !ok || doc.IsArchived {
		return nil, errors.New("document not found")
	}

	dCopy := *doc
	dCopy.FormattedTime = formatTimeAgo(dCopy.UpdatedAt)
	return &dCopy, nil
}

// CreateDocument creates a new document
func (r *Repository) CreateDocument(userID, folderID, title, status string) (*Document, error) {
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		cleanTitle = "Untitled Page"
	}

	if status == "" {
		status = "TODO"
	}

	r.mockMu.Lock()
	defer r.mockMu.Unlock()

	id := randomID("doc")
	now := time.Now().UTC()

	doc := &Document{
		ID:            id,
		Title:         cleanTitle,
		Content:       "# " + cleanTitle + "\n\nStart writing your document here...",
		Icon:          "📄",
		Status:        status,
		Position:      len(r.documents),
		UserID:        userID,
		FolderID:      folderID,
		CreatedAt:     now,
		UpdatedAt:     now,
		FormattedTime: "Just now",
	}

	r.documents[id] = doc
	return doc, nil
}

// SaveDocument updates title, content, icon, and status with autosave timestamp
func (r *Repository) SaveDocument(id, userID, title, content, icon, status string) (*Document, error) {
	r.mockMu.Lock()
	defer r.mockMu.Unlock()

	doc, ok := r.documents[id]
	if !ok || doc.IsArchived {
		return nil, errors.New("document not found")
	}

	if strings.TrimSpace(title) != "" {
		doc.Title = strings.TrimSpace(title)
	}
	doc.Content = content
	if icon != "" {
		doc.Icon = icon
	}
	if status != "" {
		doc.Status = status
	}
	doc.UpdatedAt = time.Now().UTC()
	doc.FormattedTime = "Just now"

	dCopy := *doc
	return &dCopy, nil
}

// UpdateDocumentStatus changes document Kanban status (BACKLOG, TODO, IN_PROGRESS, DONE)
func (r *Repository) UpdateDocumentStatus(id, userID, status string) (*Document, error) {
	r.mockMu.Lock()
	defer r.mockMu.Unlock()

	doc, ok := r.documents[id]
	if !ok || doc.IsArchived {
		return nil, errors.New("document not found")
	}

	doc.Status = status
	doc.UpdatedAt = time.Now().UTC()
	doc.FormattedTime = "Just now"

	dCopy := *doc
	return &dCopy, nil
}

// DeleteDocument marks a document as archived
func (r *Repository) DeleteDocument(id, userID string) error {
	r.mockMu.Lock()
	defer r.mockMu.Unlock()

	doc, ok := r.documents[id]
	if !ok {
		return errors.New("document not found")
	}

	doc.IsArchived = true
	return nil
}

// BuildKanbanColumns groups a slice of documents into standard status columns
func (r *Repository) BuildKanbanColumns(docs []*Document) []KanbanColumn {
	cols := []KanbanColumn{
		{Status: "BACKLOG", Title: "Backlog", Icon: "📥", Documents: []*Document{}},
		{Status: "TODO", Title: "To Do", Icon: "📋", Documents: []*Document{}},
		{Status: "IN_PROGRESS", Title: "In Progress", Icon: "⚡", Documents: []*Document{}},
		{Status: "DONE", Title: "Completed", Icon: "✅", Documents: []*Document{}},
	}

	statusMap := make(map[string]int)
	for idx, c := range cols {
		statusMap[c.Status] = idx
	}

	for _, d := range docs {
		idx, ok := statusMap[d.Status]
		if !ok {
			idx = 1 // default to TODO
		}
		cols[idx].Documents = append(cols[idx].Documents, d)
	}

	for i := range cols {
		cols[i].Count = len(cols[i].Documents)
	}

	return cols
}
