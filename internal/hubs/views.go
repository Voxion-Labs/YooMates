package hubs

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"yoomates/internal/auth"
)

type ServerItem struct {
	ID       string
	Name     string
	Initials string
}

type ChannelItem struct {
	ID          string
	Name        string
	Type        string
	Description string
}

type MessageItem struct {
	ID            string
	AuthorName    string
	Initials      string
	Content       string
	FileURL       string
	FileName      string
	FormattedTime string
}

type MemberItem struct {
	ID       string
	Name     string
	Username string
	Initials string
	IsOnline bool
}

type CurrentUserItem struct {
	ID       string
	Name     string
	Username string
	Initials string
}

type HubsViewModel struct {
	CurrentServer  ServerItem
	Servers        []ServerItem
	CurrentChannel ChannelItem
	Channels       []ChannelItem
	CurrentUser    CurrentUserItem
	Messages       []MessageItem
	OnlineMembers  []MemberItem
	OfflineMembers []MemberItem
}

// ViewEngine manages parsed html/templates
type ViewEngine struct {
	templates *template.Template
	pool      *pgxpool.Pool
}

// NewViewEngine parses all html templates in the templates directory
func NewViewEngine(templatesDir string, pool *pgxpool.Pool) (*ViewEngine, error) {
	pattern := filepath.Join(templatesDir, "*.html")
	partialsPattern := filepath.Join(templatesDir, "partials", "*.html")

	tmpl, err := template.ParseGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root templates: %w", err)
	}

	tmpl, err = tmpl.ParseGlob(partialsPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse partial templates: %w", err)
	}

	return &ViewEngine{
		templates: tmpl,
		pool:      pool,
	}, nil
}

// HubsHandler renders the master 3-column Hubs chat surface
func (v *ViewEngine) HubsHandler(w http.ResponseWriter, r *http.Request) {
	userCtx, _ := auth.GetUserFromContext(r.Context())

	model := HubsViewModel{
		CurrentServer: ServerItem{
			ID:       "srv_general",
			Name:     "Voxion Labs",
			Initials: "VL",
		},
		Servers: []ServerItem{
			{ID: "srv_general", Name: "Voxion Labs", Initials: "VL"},
			{ID: "srv_engineering", Name: "Engineering", Initials: "EN"},
			{ID: "srv_design", Name: "Design & UX", Initials: "DX"},
		},
		CurrentChannel: ChannelItem{
			ID:          "chan_general",
			Name:        "general",
			Type:        "TEXT",
			Description: "The central nervous system for team announcements and collaboration",
		},
		Channels: []ChannelItem{
			{ID: "chan_general", Name: "general", Type: "TEXT"},
			{ID: "chan_engineering", Name: "engineering", Type: "TEXT"},
			{ID: "chan_architecture", Name: "architecture", Type: "TEXT"},
			{ID: "chan_random", Name: "random", Type: "TEXT"},
		},
		CurrentUser: CurrentUserItem{
			ID:       "usr_guest",
			Name:     "Team Member",
			Username: "member",
			Initials: "TM",
		},
		Messages: []MessageItem{
			{
				ID:            "msg_1",
				AuthorName:    "Rudranarayan Jena",
				Initials:      "RJ",
				Content:       "Welcome to YooMates! The new Go + HTMX engine is active with zero TS/node_modules runtime overhead.",
				FormattedTime: "Today at " + time.Now().Add(-5*time.Minute).Format("15:04"),
			},
			{
				ID:            "msg_2",
				AuthorName:    "Voxion Bot",
				Initials:      "VB",
				Content:       "Real-time WebSocket hub active. Sub-phase 2.2 template layout rendered successfully.",
				FormattedTime: "Today at " + time.Now().Format("15:04"),
			},
		},
		OnlineMembers: []MemberItem{
			{ID: "usr_1", Name: "Rudranarayan Jena", Username: "rjena", Initials: "RJ", IsOnline: true},
			{ID: "usr_2", Name: "Voxion Bot", Username: "voxion-bot", Initials: "VB", IsOnline: true},
		},
		OfflineMembers: []MemberItem{
			{ID: "usr_3", Name: "Liam Brooks", Username: "liam", Initials: "LB", IsOnline: false},
		},
	}

	if userCtx != nil {
		initials := "U"
		if len(userCtx.Name) > 0 {
			initials = strings.ToUpper(string(userCtx.Name[0]))
		} else if len(userCtx.Username) > 0 {
			initials = strings.ToUpper(string(userCtx.Username[0]))
		}
		model.CurrentUser = CurrentUserItem{
			ID:       userCtx.ID,
			Name:     userCtx.Name,
			Username: userCtx.Username,
			Initials: initials,
		}
	}

	// Fetch dynamic channel if query param provided
	channelID := r.URL.Query().Get("channel_id")
	if channelID != "" {
		for _, ch := range model.Channels {
			if ch.ID == channelID {
				model.CurrentChannel = ch
				break
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := v.templates.ExecuteTemplate(w, "hubs.html", model); err != nil {
		log.Printf("[view-error] Failed to execute hubs.html: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
