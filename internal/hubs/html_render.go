package hubs

import (
	"fmt"
	"html"
	"strings"
)

// RenderAttachmentHTML generates HTML for uploaded files (image preview or download card)
func RenderAttachmentHTML(item *MessageItem) string {
	if item.FileURL == "" {
		return ""
	}
	escapedURL := html.EscapeString(item.FileURL)
	escapedName := html.EscapeString(item.FileName)
	if escapedName == "" {
		escapedName = "Attachment"
	}

	lower := strings.ToLower(item.FileURL)
	if strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".gif") ||
		strings.HasSuffix(lower, ".webp") {
		return fmt.Sprintf(`
      <div class="msg-attachment-box">
        <a href="%s" target="_blank" rel="noopener noreferrer">
          <img src="%s" alt="%s" class="msg-attachment-img" loading="lazy" />
        </a>
      </div>`, escapedURL, escapedURL, escapedName)
	}

	meta := ""
	if item.FileSize > 0 {
		if item.FileSize < 1024 {
			meta = fmt.Sprintf("%d B", item.FileSize)
		} else if item.FileSize < 1024*1024 {
			meta = fmt.Sprintf("%.1f KB", float64(item.FileSize)/1024)
		} else {
			meta = fmt.Sprintf("%.1f MB", float64(item.FileSize)/(1024*1024))
		}
	}

	return fmt.Sprintf(`
      <div class="msg-attachment-box">
        <a href="%s" target="_blank" rel="noopener noreferrer" class="msg-attachment-card">
          <span class="attachment-icon">📎</span>
          <div class="attachment-info">
            <span class="attachment-name">%s</span>
            <span class="attachment-meta">%s</span>
          </div>
          <span class="attachment-download">⬇</span>
        </a>
      </div>`, escapedURL, escapedName, meta)
}

// RenderMessageOOB generates an HTMX Out-Of-Band swap fragment that appends the message to #messages-container
func RenderMessageOOB(item *MessageItem) string {
	escapedContent := html.EscapeString(item.Content)
	escapedAuthor := html.EscapeString(item.AuthorName)
	attachmentHTML := RenderAttachmentHTML(item)

	bodyHTML := ""
	if escapedContent != "" {
		bodyHTML = fmt.Sprintf(`<div class="msg-body">%s</div>`, escapedContent)
	}

	return fmt.Sprintf(`
<div id="messages-container" hx-swap-oob="beforeend">
  <article class="message-item" id="msg-%s">
    <div class="msg-avatar">%s</div>
    <div class="msg-content-box">
      <header class="msg-meta">
        <span class="msg-author">%s</span>
        <time class="msg-time">%s</time>
      </header>
      %s
      %s
      <div id="reactions-msg-%s" class="msg-reactions"></div>
    </div>
    <div class="msg-actions">
      <button class="action-btn" title="React Thumbs Up" ws-send hx-vals='{"type":"reaction","message_id":"%s","emoji":"👍"}'>👍</button>
      <button class="action-btn" title="React Heart" ws-send hx-vals='{"type":"reaction","message_id":"%s","emoji":"❤️"}'>❤️</button>
      <button class="action-btn" title="React Fire" ws-send hx-vals='{"type":"reaction","message_id":"%s","emoji":"🔥"}'>🔥</button>
      <button class="action-btn" title="Delete Message" ws-send hx-vals='{"type":"delete","message_id":"%s"}'>🗑</button>
    </div>
  </article>
</div>
`, item.ID, item.Initials, escapedAuthor, item.FormattedTime, bodyHTML, attachmentHTML, item.ID, item.ID, item.ID, item.ID, item.ID)
}

// RenderMessageEditOOB generates an HTMX OOB swap fragment that replaces the edited message element
func RenderMessageEditOOB(item *MessageItem) string {
	escapedContent := html.EscapeString(item.Content)

	return fmt.Sprintf(`
<div id="msg-%s" hx-swap-oob="outerHTML">
  <article class="message-item" id="msg-%s">
    <div class="msg-avatar">%s</div>
    <div class="msg-content-box">
      <header class="msg-meta">
        <span class="msg-author">%s</span>
        <time class="msg-time">%s (edited)</time>
      </header>
      <div class="msg-body">%s</div>
      <div id="reactions-msg-%s" class="msg-reactions"></div>
    </div>
  </article>
</div>
`, item.ID, item.ID, item.Initials, html.EscapeString(item.AuthorName), item.FormattedTime, escapedContent, item.ID)
}

// RenderMessageDeleteOOB generates an HTMX OOB swap fragment indicating message deletion
func RenderMessageDeleteOOB(messageID string) string {
	return fmt.Sprintf(`
<div id="msg-%s" hx-swap-oob="outerHTML">
  <article class="message-item msg-deleted" id="msg-%s">
    <div class="msg-body" style="color: var(--text-muted); font-style: italic; font-size: 12.5px;">
      [This message was deleted]
    </div>
  </article>
</div>
`, messageID, messageID)
}

// RenderTypingOOB renders the typing status banner
func RenderTypingOOB(username string, isTyping bool) string {
	if !isTyping || username == "" {
		return `<div id="typing-indicator" hx-swap-oob="innerHTML"></div>`
	}
	return fmt.Sprintf(`
<div id="typing-indicator" hx-swap-oob="innerHTML">
  <span class="typing-text"><span class="typing-dot"></span> %s is typing...</span>
</div>
`, html.EscapeString(username))
}

// RenderReactionsOOB renders the reaction count badges
func RenderReactionsOOB(messageID string, reactions []ReactionItem) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<div id="reactions-msg-%s" class="msg-reactions" hx-swap-oob="innerHTML">`, messageID))
	for _, r := range reactions {
		sb.WriteString(fmt.Sprintf(`
      <span class="reaction-pill">%s %d</span>
    `, r.Emoji, r.Count))
	}
	sb.WriteString(`</div>`)
	return sb.String()
}
