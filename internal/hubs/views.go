package hubs

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

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
	FileType      string
	FileSize      int
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
	VoiceChannels  []ChannelItem
	CurrentUser    CurrentUserItem
	Messages       []MessageItem
	OnlineMembers  []MemberItem
	OfflineMembers []MemberItem
}

// ViewEngine manages parsed html/templates
type ViewEngine struct {
	templates *template.Template
	db        *sql.DB
}

// NewViewEngine parses all html templates in the templates directory
func NewViewEngine(templatesDir string, db *sql.DB) (*ViewEngine, error) {
	funcMap := template.FuncMap{
		"isImage": func(url string) bool {
			lower := strings.ToLower(url)
			return strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") ||
				strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".gif") ||
				strings.HasSuffix(lower, ".webp")
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
		"formatPercent": func(p float64) string {
			return fmt.Sprintf("%.1f%%", p)
		},
	}

	basePattern := filepath.Join(templatesDir, "base.html")
	pagePattern := filepath.Join(templatesDir, "hubs.html")
	partialsPattern := filepath.Join(templatesDir, "partials", "*.html")

	tmpl := template.New("").Funcs(funcMap)
	tmpl, err := tmpl.ParseFiles(basePattern, pagePattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base and hubs templates: %w", err)
	}

	tmpl, err = tmpl.ParseGlob(partialsPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse partial templates: %w", err)
	}

	return &ViewEngine{
		templates: tmpl,
		db:        db,
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
		VoiceChannels: []ChannelItem{
			{ID: "chan_voice_lounge", Name: "Lounge", Type: "VOICE", Description: "Casual voice chat room"},
			{ID: "chan_voice_standup", Name: "Daily Standup", Type: "VOICE", Description: "Engineering sync room"},
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
