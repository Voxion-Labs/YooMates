package feed

import (
	"time"
)

// PollOption represents an individual selectable option in a post's poll
type PollOption struct {
	ID         string  `json:"id"`
	PollID     string  `json:"poll_id"`
	Text       string  `json:"text"`
	Position   int     `json:"position"`
	VoteCount  int     `json:"vote_count"`
	Percentage float64 `json:"percentage"`
}

// Poll represents an interactive poll attached to a post
type Poll struct {
	ID                 string       `json:"id"`
	PostID             string       `json:"post_id"`
	Question           string       `json:"question"`
	AllowMultipleVotes bool         `json:"allow_multiple_votes"`
	ExpiresAt          *time.Time   `json:"expires_at,omitempty"`
	Options            []PollOption `json:"options"`
	TotalVotes         int          `json:"total_votes"`
	HasVoted           bool         `json:"has_voted"`
	UserVotedOptionIDs []string     `json:"user_voted_option_ids"`
}

// Comment represents a reply to a post
type Comment struct {
	ID             string    `json:"id"`
	PostID         string    `json:"post_id"`
	Text           string    `json:"text"`
	AuthorID       string    `json:"author_id"`
	AuthorName     string    `json:"author_name"`
	AuthorUsername string    `json:"author_username"`
	AuthorInitials string    `json:"author_initials"`
	CreatedAt      time.Time `json:"created_at"`
	FormattedTime  string    `json:"formatted_time"`
}

// Post represents a publication in the Global Feed
type Post struct {
	ID             string    `json:"id"`
	Content        string    `json:"content"`
	MediaURL       string    `json:"media_url,omitempty"`
	MediaType      string    `json:"media_type,omitempty"`
	MediaName      string    `json:"media_name,omitempty"`
	AuthorID       string    `json:"author_id"`
	AuthorName     string    `json:"author_name"`
	AuthorUsername string    `json:"author_username"`
	AuthorInitials string    `json:"author_initials"`
	IsArchived     bool      `json:"is_archived"`
	QuotePostID    *string   `json:"quote_post_id,omitempty"`
	QuotePost      *Post     `json:"quote_post,omitempty"`
	LikesCount     int       `json:"likes_count"`
	HasLiked       bool      `json:"has_liked"`
	CommentsCount  int       `json:"comments_count"`
	Poll           *Poll     `json:"poll,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	FormattedTime  string    `json:"formatted_time"`
}

// CurrentUserItem provides basic identity for the active session
type CurrentUserItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Initials string `json:"initials"`
}

// TrendingItem describes a trending topic widget in the right rail
type TrendingItem struct {
	Category  string `json:"category"`
	Topic     string `json:"topic"`
	PostCount string `json:"post_count"`
}

// FollowSuggestion describes suggested accounts to follow
type FollowSuggestion struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Initials string `json:"initials"`
	Bio      string `json:"bio"`
}

// FeedViewModel represents the full page template model for /global
type FeedViewModel struct {
	CurrentUser    CurrentUserItem
	Posts          []*Post
	NextCursor     string
	HasMore        bool
	TrendingTopics []TrendingItem
	WhoToFollow    []FollowSuggestion
}

// CreatePollInput holds input options when attaching a poll to a new post
type CreatePollInput struct {
	Question string
	Options  []string
}
