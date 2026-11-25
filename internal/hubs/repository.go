package hubs

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ReactionItem struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func generateID(prefix string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

// SaveMessage stores a new chat message into MySQL
func (r *Repository) SaveMessage(ctx context.Context, authorID, authorName, channelID, serverID, content string) (*MessageItem, error) {
	return r.SaveMessageWithAttachment(ctx, authorID, authorName, channelID, serverID, content, "", "", "", 0)
}

// SaveMessageWithAttachment stores a chat message with optional attachment details into MySQL
func (r *Repository) SaveMessageWithAttachment(ctx context.Context, authorID, authorName, channelID, serverID, content, fileURL, fileName, fileType string, fileSize int) (*MessageItem, error) {
	msgID := generateID("msg")
	now := time.Now()

	initials := "U"
	if len(authorName) > 0 {
		initials = strings.ToUpper(string(authorName[0]))
	}

	item := &MessageItem{
		ID:            msgID,
		AuthorName:    authorName,
		Initials:      initials,
		Content:       content,
		FileURL:       fileURL,
		FileName:      fileName,
		FileType:      fileType,
		FileSize:      fileSize,
		FormattedTime: "Today at " + now.Format("15:04"),
	}

	if r.db == nil {
		// Mock fallback if running without live database
		return item, nil
	}

	// Look up member_id for this user and server
	var memberID string
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM members WHERE user_id = ? AND server_id = ? LIMIT 1
	`, authorID, serverID).Scan(&memberID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Auto-create membership if not exists
			memberID = generateID("mem")
			_, _ = r.db.ExecContext(ctx, `
				INSERT INTO members (id, role, user_id, server_id) VALUES (?, 'MEMBER', ?, ?)
			`, memberID, authorID, serverID)
		} else {
			memberID = generateID("mem")
		}
	}

	query := `
		INSERT INTO messages (id, content, file_url, file_name, file_type, file_size, member_id, user_id, channel_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.ExecContext(ctx, query, msgID, content, fileURL, fileName, fileType, fileSize, memberID, authorID, channelID, now, now)
	if err != nil {
		return item, fmt.Errorf("failed to persist message to mysql: %w", err)
	}

	return item, nil
}

// EditMessage updates an existing message in MySQL
func (r *Repository) EditMessage(ctx context.Context, messageID, authorID, newContent string) (*MessageItem, error) {
	now := time.Now()
	item := &MessageItem{
		ID:            messageID,
		Content:       newContent,
		FormattedTime: "Today at " + now.Format("15:04") + " (edited)",
	}

	if r.db == nil {
		return item, nil
	}

	query := `
		UPDATE messages 
		SET content = ?, is_edited = TRUE, updated_at = ? 
		WHERE id = ? AND user_id = ?
	`
	res, err := r.db.ExecContext(ctx, query, newContent, now, messageID, authorID)
	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, errors.New("message not found or unauthorized to edit")
	}

	return item, nil
}

// DeleteMessage soft-deletes a message in MySQL
func (r *Repository) DeleteMessage(ctx context.Context, messageID, authorID string) error {
	if r.db == nil {
		return nil
	}

	query := `
		UPDATE messages 
		SET deleted = TRUE, content = '[This message was deleted]', updated_at = ? 
		WHERE id = ? AND user_id = ?
	`
	res, err := r.db.ExecContext(ctx, query, time.Now(), messageID, authorID)
	if err != nil {
		return fmt.Errorf("failed to soft delete message: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("message not found or unauthorized to delete")
	}

	return nil
}

// ToggleReaction toggles an emoji reaction on a message
func (r *Repository) ToggleReaction(ctx context.Context, messageID, authorID, serverID, emoji string) ([]ReactionItem, error) {
	if r.db == nil {
		return []ReactionItem{{Emoji: emoji, Count: 1}}, nil
	}

	// Find member ID
	var memberID string
	_ = r.db.QueryRowContext(ctx, `SELECT id FROM members WHERE user_id = ? AND server_id = ? LIMIT 1`, authorID, serverID).Scan(&memberID)
	if memberID == "" {
		memberID = generateID("mem")
	}

	// Check if already reacted
	var existingID string
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM reactions WHERE message_id = ? AND user_id = ? AND emoji = ? LIMIT 1
	`, messageID, authorID, emoji).Scan(&existingID)

	if err == nil && existingID != "" {
		// Remove reaction
		_, _ = r.db.ExecContext(ctx, `DELETE FROM reactions WHERE id = ?`, existingID)
	} else {
		// Add reaction
		reactID := generateID("react")
		_, _ = r.db.ExecContext(ctx, `
			INSERT INTO reactions (id, emoji, message_id, member_id, user_id)
			VALUES (?, ?, ?, ?, ?)
		`, reactID, emoji, messageID, memberID, authorID)
	}

	// Fetch aggregated reactions count
	rows, err := r.db.QueryContext(ctx, `
		SELECT emoji, COUNT(*) as cnt 
		FROM reactions 
		WHERE message_id = ? 
		GROUP BY emoji
	`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []ReactionItem
	for rows.Next() {
		var item ReactionItem
		if err := rows.Scan(&item.Emoji, &item.Count); err == nil {
			reactions = append(reactions, item)
		}
	}

	return reactions, nil
}
