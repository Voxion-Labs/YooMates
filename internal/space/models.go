package space

import (
	"time"
)

// DocumentFolder represents a folder/workspace category for documents
type DocumentFolder struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	Position  int       `json:"position"`
	UserID    string    `json:"user_id"`
	DocCount  int       `json:"doc_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Document represents a markdown document or workspace item
type Document struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	Icon          string    `json:"icon"`
	Status        string    `json:"status"` // BACKLOG, TODO, IN_PROGRESS, DONE
	Position      int       `json:"position"`
	UserID        string    `json:"user_id"`
	FolderID      string    `json:"folder_id"`
	ParentID      *string   `json:"parent_id,omitempty"`
	IsArchived    bool      `json:"is_archived"`
	IsPublished   bool      `json:"is_published"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	FormattedTime string    `json:"formatted_time"`
}

// KanbanColumn represents a grouping column in the board view
type KanbanColumn struct {
	Status    string      `json:"status"`
	Title     string      `json:"title"`
	Icon      string      `json:"icon"`
	Documents []*Document `json:"documents"`
	Count     int         `json:"count"`
}

// CurrentUserItem provides basic session user identity
type CurrentUserItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Initials string `json:"initials"`
}

// SpaceViewModel is the top-level payload for the My Space workspace surface
type SpaceViewModel struct {
	CurrentUser    CurrentUserItem
	Folders        []*DocumentFolder
	CurrentFolder  *DocumentFolder
	ActiveDocument *Document
	ViewType       string // "editor", "kanban", "list"
	Documents      []*Document
	KanbanColumns  []KanbanColumn
	TotalDocs      int
}
