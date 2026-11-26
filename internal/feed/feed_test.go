package feed

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestFeedRepository_InitialSeed(t *testing.T) {
	repo := NewRepository(nil)

	posts, nextCursor, err := repo.GetPosts("", 10, "")
	if err != nil {
		t.Fatalf("unexpected error fetching seed posts: %v", err)
	}

	if len(posts) < 3 {
		t.Fatalf("expected at least 3 seed posts, got %d", len(posts))
	}

	// Verify quote post is linked
	var quotePostFound bool
	for _, p := range posts {
		if p.QuotePost != nil {
			quotePostFound = true
			if p.QuotePost.AuthorName == "" {
				t.Errorf("expected quote post to have author name")
			}
		}
	}
	if !quotePostFound {
		t.Errorf("expected at least one post with quote post reference")
	}

	_ = nextCursor
}

func TestFeedRepository_CreatePost(t *testing.T) {
	repo := NewRepository(nil)

	post, err := repo.CreatePost("usr_test", "Alice", "alice", "Testing pure Go HTMX feed", "", "", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error creating post: %v", err)
	}

	if post.ID == "" {
		t.Errorf("expected post to have an ID")
	}
	if post.AuthorName != "Alice" {
		t.Errorf("expected author Alice, got %s", post.AuthorName)
	}
	if post.AuthorInitials != "A" {
		t.Errorf("expected initials A, got %s", post.AuthorInitials)
	}

	// Verify it shows up in GetPosts
	posts, _, err := repo.GetPosts("", 10, "usr_test")
	if err != nil {
		t.Fatalf("unexpected error fetching posts: %v", err)
	}
	if len(posts) == 0 || posts[0].ID != post.ID {
		t.Errorf("expected newly created post to be first in chronological feed")
	}
}

func TestFeedRepository_CursorPagination(t *testing.T) {
	repo := NewRepository(nil)

	// Fetch first page of 2 posts
	page1, cursor1, err := repo.GetPosts("", 2, "")
	if err != nil {
		t.Fatalf("unexpected error on page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("expected page 1 to have 2 posts, got %d", len(page1))
	}
	if cursor1 == "" {
		t.Fatalf("expected non-empty cursor for next page")
	}

	// Fetch second page using cursor
	page2, _, err := repo.GetPosts(cursor1, 2, "")
	if err != nil {
		t.Fatalf("unexpected error on page 2: %v", err)
	}
	if len(page2) == 0 {
		t.Fatalf("expected page 2 to have posts")
	}
	if page2[0].ID == page1[0].ID || page2[0].ID == page1[1].ID {
		t.Errorf("page 2 post overlapped with page 1")
	}
}

func TestFeedRepository_LikeToggle(t *testing.T) {
	repo := NewRepository(nil)

	// Like post
	hasLiked, count, err := repo.ToggleLike("post_launch", "usr_tester")
	if err != nil {
		t.Fatalf("unexpected error liking post: %v", err)
	}
	if !hasLiked {
		t.Errorf("expected hasLiked to be true")
	}
	if count <= 0 {
		t.Errorf("expected like count to increase")
	}

	// Unlike post
	hasLiked, count2, err := repo.ToggleLike("post_launch", "usr_tester")
	if err != nil {
		t.Fatalf("unexpected error unliking post: %v", err)
	}
	if hasLiked {
		t.Errorf("expected hasLiked to be false on second toggle")
	}
	if count2 != count-1 {
		t.Errorf("expected like count to decrement by 1, got %d vs %d", count2, count)
	}
}

func TestFeedRepository_Comments(t *testing.T) {
	repo := NewRepository(nil)

	cmt, err := repo.AddComment("post_launch", "usr_bob", "Bob", "bob", "Great progress!")
	if err != nil {
		t.Fatalf("unexpected error adding comment: %v", err)
	}
	if cmt.Text != "Great progress!" {
		t.Errorf("expected comment text 'Great progress!', got %s", cmt.Text)
	}

	comments, err := repo.GetComments("post_launch")
	if err != nil {
		t.Fatalf("unexpected error getting comments: %v", err)
	}
	if len(comments) < 2 {
		t.Errorf("expected at least 2 comments, got %d", len(comments))
	}
}

func TestFeedRepository_PollVoting(t *testing.T) {
	repo := NewRepository(nil)

	poll, err := repo.VotePoll("poll_engine", "opt_1", "usr_new_voter")
	if err != nil {
		t.Fatalf("unexpected error voting in poll: %v", err)
	}

	if !poll.HasVoted {
		t.Errorf("expected poll.HasVoted to be true")
	}

	// Second vote by same user should be ignored
	totalBefore := poll.TotalVotes
	poll2, err := repo.VotePoll("poll_engine", "opt_2", "usr_new_voter")
	if err != nil {
		t.Fatalf("unexpected error on second vote: %v", err)
	}
	if poll2.TotalVotes != totalBefore {
		t.Errorf("expected total votes to remain %d, got %d", totalBefore, poll2.TotalVotes)
	}
}

func TestViewEngine_RenderGlobal(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "web", "templates")
	repo := NewRepository(nil)
	engine, err := NewViewEngine(templatesDir, repo)
	if err != nil {
		t.Fatalf("failed to create ViewEngine: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/global", nil)
	rr := httptest.NewRecorder()

	engine.GlobalHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "feed-layout") {
		t.Error("expected body to contain feed-layout class")
	}
	if !strings.Contains(body, "posts-stream-container") {
		t.Error("expected body to contain posts-stream-container")
	}
	if !strings.Contains(body, "feed-composer-box") {
		t.Error("expected body to contain feed-composer-box")
	}
	if !strings.Contains(body, "Global Feed") {
		t.Error("expected body to contain 'Global Feed'")
	}
}

func TestViewEngine_PostsStreamHandler(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "web", "templates")
	repo := NewRepository(nil)
	engine, err := NewViewEngine(templatesDir, repo)
	if err != nil {
		t.Fatalf("failed to create ViewEngine: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/global/posts?limit=2", nil)
	rr := httptest.NewRecorder()

	engine.PostsStreamHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "feed-post-card") {
		t.Error("expected response to contain feed-post-card")
	}
}

func TestViewEngine_CreatePostAndLike(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "web", "templates")
	repo := NewRepository(nil)
	engine, err := NewViewEngine(templatesDir, repo)
	if err != nil {
		t.Fatalf("failed to create ViewEngine: %v", err)
	}

	// Create Post
	form := url.Values{}
	form.Set("content", "Hello from automated test")
	req := httptest.NewRequest(http.MethodPost, "/global/posts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	engine.CreatePostHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Hello from automated test") {
		t.Error("expected rendered post to contain new content")
	}

	// Like Post
	likeReq := httptest.NewRequest(http.MethodPost, "/global/posts/post_launch/like", nil)
	likeRR := httptest.NewRecorder()
	engine.LikeHandler(likeRR, likeReq)
	if likeRR.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", likeRR.Code)
	}
	likeBody := likeRR.Body.String()
	if !strings.Contains(likeBody, "like-btn") {
		t.Error("expected like button in response")
	}
}
