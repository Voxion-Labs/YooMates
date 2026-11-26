package feed

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

// Repository manages persistence for the Global Feed (MySQL + In-Memory Fallback)
type Repository struct {
	db       *sql.DB
	mockMu   sync.RWMutex
	posts    map[string]*Post
	comments map[string][]*Comment
	likes    map[string]map[string]bool // post_id -> user_id -> true
	polls    map[string]*Poll           // poll_id -> Poll
	votes    map[string]map[string]bool // option_id -> user_id -> true
}

func randomID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

func formatPostTime(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "Just now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh", int(diff.Hours()))
	} else if diff < 7*24*time.Hour {
		return fmt.Sprintf("%dd", int(diff.Hours()/24))
	}
	return t.Format("Jan 02")
}

// NewRepository creates a new Feed repository
func NewRepository(db *sql.DB) *Repository {
	repo := &Repository{
		db:       db,
		posts:    make(map[string]*Post),
		comments: make(map[string][]*Comment),
		likes:    make(map[string]map[string]bool),
		polls:    make(map[string]*Poll),
		votes:    make(map[string]map[string]bool),
	}

	// Seed realistic demo posts for initial rendering and zero-db offline environments
	repo.seedInitialData()
	return repo
}

func (r *Repository) seedInitialData() {
	now := time.Now().UTC()

	// 1. Initial Launch Post
	p1 := &Post{
		ID:             "post_launch",
		Content:        "Welcome to the new YooMates Global Feed. Engineered in pure Go + HTMX with zero TypeScript/node_modules runtime overhead and <5ms TTFB. Monochromatic, high-density, real-time.",
		AuthorID:       "usr_rjena",
		AuthorName:     "Rudranarayan Jena",
		AuthorUsername: "rjena",
		AuthorInitials: "RJ",
		LikesCount:     12,
		CommentsCount:  3,
		CreatedAt:      now.Add(-2 * time.Hour),
		FormattedTime:  "2h",
	}

	// 2. Poll Post
	p2ID := "post_poll"
	pollID := "poll_engine"
	poll := &Poll{
		ID:       pollID,
		PostID:   p2ID,
		Question: "Which backend architecture do you prefer for real-time collaboration?",
		Options: []PollOption{
			{ID: "opt_1", PollID: pollID, Text: "Go + HTMX + Dumb-Pipe WASM", Position: 0, VoteCount: 38, Percentage: 76.0},
			{ID: "opt_2", PollID: pollID, Text: "Next.js + Prisma + Pusher", Position: 1, VoteCount: 4, Percentage: 8.0},
			{ID: "opt_3", PollID: pollID, Text: "Elixir / Phoenix LiveView", Position: 2, VoteCount: 8, Percentage: 16.0},
		},
		TotalVotes: 50,
		HasVoted:   false,
	}

	p2 := &Post{
		ID:             p2ID,
		Content:        "Architectural benchmark results are in. We dropped 400MB of node_modules and achieved 0ms offline loads with IndexedDB and WebAssembly.",
		AuthorID:       "usr_voxion",
		AuthorName:     "Voxion Bot",
		AuthorUsername: "voxion-bot",
		AuthorInitials: "VB",
		LikesCount:     27,
		CommentsCount:  5,
		Poll:           poll,
		CreatedAt:      now.Add(-45 * time.Minute),
		FormattedTime:  "45m",
	}

	// 3. Quote Post referencing p1
	p3 := &Post{
		ID:             "post_quote",
		Content:        "Notice how smooth the instant page transitions feel without client-side React hydration lag. Pure hypermedia in action.",
		AuthorID:       "usr_liam",
		AuthorName:     "Liam Brooks",
		AuthorUsername: "liambrooks",
		AuthorInitials: "LB",
		QuotePostID:    &p1.ID,
		QuotePost:      p1,
		LikesCount:     8,
		CommentsCount:  1,
		CreatedAt:      now.Add(-10 * time.Minute),
		FormattedTime:  "10m",
	}

	r.posts[p1.ID] = p1
	r.posts[p2.ID] = p2
	r.posts[p3.ID] = p3
	r.polls[poll.ID] = poll

	r.comments[p1.ID] = []*Comment{
		{
			ID:             "cmt_1",
			PostID:         p1.ID,
			Text:           "Loving the monochromatic dark aesthetics. Extremely fast.",
			AuthorID:       "usr_liam",
			AuthorName:     "Liam Brooks",
			AuthorUsername: "liambrooks",
			AuthorInitials: "LB",
			CreatedAt:      now.Add(-1 * time.Hour),
			FormattedTime:  "1h",
		},
	}
}

// GetPosts returns a chronologically sorted page of posts with cursor pagination
func (r *Repository) GetPosts(cursor string, limit int, currentUserID string) ([]*Post, string, error) {
	if limit <= 0 {
		limit = 10
	}

	r.mockMu.RLock()
	defer r.mockMu.RUnlock()

	var allPosts []*Post
	for _, p := range r.posts {
		if !p.IsArchived {
			allPosts = append(allPosts, p)
		}
	}

	// Sort newest first
	sort.Slice(allPosts, func(i, j int) bool {
		return allPosts[i].CreatedAt.After(allPosts[j].CreatedAt)
	})

	startIndex := 0
	if cursor != "" {
		for i, p := range allPosts {
			if p.ID == cursor {
				startIndex = i + 1
				break
			}
		}
	}

	if startIndex >= len(allPosts) {
		return []*Post{}, "", nil
	}

	endIndex := startIndex + limit
	if endIndex > len(allPosts) {
		endIndex = len(allPosts)
	}

	page := make([]*Post, endIndex-startIndex)
	for i := startIndex; i < endIndex; i++ {
		pCopy := *allPosts[i]
		pCopy.FormattedTime = formatPostTime(pCopy.CreatedAt)

		// Set user-specific state
		if currentUserID != "" {
			if postLikes, ok := r.likes[pCopy.ID]; ok {
				pCopy.HasLiked = postLikes[currentUserID]
			}
			if pCopy.Poll != nil {
				pollCopy := *pCopy.Poll
				pollCopy.HasVoted = false
				pollCopy.UserVotedOptionIDs = nil
				for _, opt := range pollCopy.Options {
					if optVotes, ok := r.votes[opt.ID]; ok && optVotes[currentUserID] {
						pollCopy.HasVoted = true
						pollCopy.UserVotedOptionIDs = append(pollCopy.UserVotedOptionIDs, opt.ID)
					}
				}
				pCopy.Poll = &pollCopy
			}
		}

		page[i-startIndex] = &pCopy
	}

	nextCursor := ""
	if endIndex < len(allPosts) {
		nextCursor = allPosts[endIndex-1].ID
	}

	return page, nextCursor, nil
}

// CreatePost creates a new post with optional quote, media, or poll
func (r *Repository) CreatePost(authorID, authorName, authorUsername, content, mediaURL, mediaType, mediaName, quotePostID string, pollInput *CreatePollInput) (*Post, error) {
	if strings.TrimSpace(content) == "" && mediaURL == "" && pollInput == nil {
		return nil, errors.New("post content, media or poll must not be empty")
	}

	r.mockMu.Lock()
	defer r.mockMu.Unlock()

	postID := randomID("post")
	initials := "U"
	if len(authorName) > 0 {
		initials = strings.ToUpper(string(authorName[0]))
	} else if len(authorUsername) > 0 {
		initials = strings.ToUpper(string(authorUsername[0]))
	}

	post := &Post{
		ID:             postID,
		Content:        strings.TrimSpace(content),
		MediaURL:       mediaURL,
		MediaType:      mediaType,
		MediaName:      mediaName,
		AuthorID:       authorID,
		AuthorName:     authorName,
		AuthorUsername: authorUsername,
		AuthorInitials: initials,
		LikesCount:     0,
		CommentsCount:  0,
		CreatedAt:      time.Now().UTC(),
		FormattedTime:  "Just now",
	}

	if quotePostID != "" {
		post.QuotePostID = &quotePostID
		if qp, ok := r.posts[quotePostID]; ok {
			post.QuotePost = qp
		}
	}

	// Handle Poll creation
	if pollInput != nil && len(pollInput.Options) >= 2 {
		pollID := randomID("poll")
		poll := &Poll{
			ID:                 pollID,
			PostID:             postID,
			Question:           strings.TrimSpace(pollInput.Question),
			AllowMultipleVotes: false,
			Options:            make([]PollOption, 0, len(pollInput.Options)),
			TotalVotes:         0,
			HasVoted:           false,
		}
		for idx, optText := range pollInput.Options {
			cleanText := strings.TrimSpace(optText)
			if cleanText == "" {
				continue
			}
			optID := fmt.Sprintf("%s_opt_%d", pollID, idx)
			poll.Options = append(poll.Options, PollOption{
				ID:         optID,
				PollID:     pollID,
				Text:       cleanText,
				Position:   idx,
				VoteCount:  0,
				Percentage: 0.0,
			})
		}
		if len(poll.Options) >= 2 {
			post.Poll = poll
			r.polls[pollID] = poll
		}
	}

	r.posts[postID] = post
	return post, nil
}

// ToggleLike toggles like state for a user and returns (hasLiked, totalLikes, error)
func (r *Repository) ToggleLike(postID, userID string) (bool, int, error) {
	if postID == "" || userID == "" {
		return false, 0, errors.New("postID and userID must not be empty")
	}

	r.mockMu.Lock()
	defer r.mockMu.Unlock()

	post, ok := r.posts[postID]
	if !ok {
		return false, 0, errors.New("post not found")
	}

	if r.likes[postID] == nil {
		r.likes[postID] = make(map[string]bool)
	}

	hasLiked := r.likes[postID][userID]
	if hasLiked {
		delete(r.likes[postID], userID)
		if post.LikesCount > 0 {
			post.LikesCount--
		}
		return false, post.LikesCount, nil
	}

	r.likes[postID][userID] = true
	post.LikesCount++
	return true, post.LikesCount, nil
}

// AddComment adds a comment to a post
func (r *Repository) AddComment(postID, authorID, authorName, authorUsername, text string) (*Comment, error) {
	cleanText := strings.TrimSpace(text)
	if cleanText == "" {
		return nil, errors.New("comment text must not be empty")
	}

	r.mockMu.Lock()
	defer r.mockMu.Unlock()

	post, ok := r.posts[postID]
	if !ok {
		return nil, errors.New("post not found")
	}

	initials := "U"
	if len(authorName) > 0 {
		initials = strings.ToUpper(string(authorName[0]))
	} else if len(authorUsername) > 0 {
		initials = strings.ToUpper(string(authorUsername[0]))
	}

	comment := &Comment{
		ID:             randomID("cmt"),
		PostID:         postID,
		Text:           cleanText,
		AuthorID:       authorID,
		AuthorName:     authorName,
		AuthorUsername: authorUsername,
		AuthorInitials: initials,
		CreatedAt:      time.Now().UTC(),
		FormattedTime:  "Just now",
	}

	r.comments[postID] = append(r.comments[postID], comment)
	post.CommentsCount = len(r.comments[postID])
	return comment, nil
}

// GetComments returns all comments for a post
func (r *Repository) GetComments(postID string) ([]*Comment, error) {
	r.mockMu.RLock()
	defer r.mockMu.RUnlock()

	cmts := r.comments[postID]
	result := make([]*Comment, len(cmts))
	for i, c := range cmts {
		cCopy := *c
		cCopy.FormattedTime = formatPostTime(cCopy.CreatedAt)
		result[i] = &cCopy
	}
	return result, nil
}

// VotePoll records a user's vote on a poll option and recalculates percentages
func (r *Repository) VotePoll(pollID, optionID, userID string) (*Poll, error) {
	if pollID == "" || optionID == "" || userID == "" {
		return nil, errors.New("pollID, optionID, and userID must not be empty")
	}

	r.mockMu.Lock()
	defer r.mockMu.Unlock()

	poll, ok := r.polls[pollID]
	if !ok {
		return nil, errors.New("poll not found")
	}

	if r.votes[optionID] == nil {
		r.votes[optionID] = make(map[string]bool)
	}

	// Check if already voted
	for _, opt := range poll.Options {
		if r.votes[opt.ID] != nil && r.votes[opt.ID][userID] {
			return poll, nil // Already voted
		}
	}

	// Record vote
	r.votes[optionID][userID] = true

	// Recalculate counts
	total := 0
	for i := range poll.Options {
		opt := &poll.Options[i]
		if opt.ID == optionID {
			opt.VoteCount++
		}
		total += opt.VoteCount
	}
	poll.TotalVotes = total
	poll.HasVoted = true
	poll.UserVotedOptionIDs = []string{optionID}

	// Recalculate percentages
	for i := range poll.Options {
		opt := &poll.Options[i]
		if total > 0 {
			opt.Percentage = (float64(opt.VoteCount) / float64(total)) * 100.0
		} else {
			opt.Percentage = 0.0
		}
	}

	return poll, nil
}
