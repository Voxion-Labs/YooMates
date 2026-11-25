// YooMates Dumb Pipe JS - Zero Framework, Zero State (<110 lines)
// Strictly an event, offline cache, and DOM transport layer.

(function () {
  'use strict';

  function scrollToBottom() {
    const stream = document.getElementById('message-stream');
    if (stream) {
      stream.scrollTop = stream.scrollHeight;
    }
  }

  // 1. Service Worker Registration for Offline Caching
  if ('serviceWorker' in navigator) {
    window.addEventListener('load', function () {
      navigator.serviceWorker.register('/sw.js').catch(function (err) {
        console.warn('[sw] Registration skipped or failed:', err);
      });
    });
  }

  // 2. Network Status Badge Updates
  function updateNetworkStatus() {
    const badge = document.getElementById('network-status-badge');
    if (!badge) return;
    if (navigator.onLine) {
      badge.textContent = 'WS: Connected';
      badge.style.color = 'var(--text-muted)';
      badge.style.borderColor = 'var(--border-muted)';
    } else {
      badge.textContent = 'Offline (Cached)';
      badge.style.color = 'var(--accent-red)';
      badge.style.borderColor = 'var(--accent-red)';
    }
  }
  window.addEventListener('online', updateNetworkStatus);
  window.addEventListener('offline', updateNetworkStatus);

  // 3. 0ms Instant Cache Hydration & DOM Sync
  function initOfflineCache() {
    const channelInput = document.querySelector('input[name="channel_id"]');
    const channelId = channelInput ? channelInput.value : 'chan_general';
    if (!window.YooCache) return;

    window.YooCache.syncDOMToCache(channelId);

    const container = document.getElementById('messages-container');
    if (container && container.querySelectorAll('.message-item').length === 0) {
      window.YooCache.getChannelMessages(channelId).then(function (messages) {
        if (!messages || messages.length === 0) return;
        messages.forEach(function (m) {
          const art = document.createElement('article');
          art.className = 'message-item';
          art.id = 'msg-' + m.id;
          art.innerHTML =
            '<div class="msg-avatar">' + (m.initials || 'U') + '</div>' +
            '<div class="msg-content-box">' +
              '<header class="msg-meta">' +
                '<span class="msg-author">' + (m.author_name || 'Member') + '</span>' +
                '<time class="msg-time">' + (m.time || '') + '</time>' +
              '</header>' +
              '<div class="msg-body">' + (m.content || '') + '</div>' +
            '</div>';
          container.appendChild(art);
        });
        scrollToBottom();
      });
    }
  }

  // 4. Scroll & Sync on Load and HTMX Swaps
  window.addEventListener('DOMContentLoaded', function () {
    scrollToBottom();
    initOfflineCache();
    updateNetworkStatus();
  });

  document.addEventListener('htmx:afterSwap', function (evt) {
    if (evt.target && (evt.target.id === 'message-stream' || evt.target.closest('#message-stream') || evt.target.id === 'messages-container')) {
      scrollToBottom();
      const channelInput = document.querySelector('input[name="channel_id"]');
      const channelId = channelInput ? channelInput.value : 'chan_general';
      if (window.YooCache) {
        window.YooCache.syncDOMToCache(channelId);
      }
    }
  });

  // 5. Clear chat input after HTMX WebSocket sends message
  document.addEventListener('htmx:wsAfterSend', function () {
    const input = document.getElementById('chat-input-field');
    if (input) {
      input.value = '';
      input.focus();
    }
  });

  // 6. Attachment upload and preview handling
  function initAttachmentPipeline() {
    const attachBtn = document.getElementById('chat-attach-trigger');
    const fileInput = document.getElementById('chat-file-input');
    const previewBar = document.getElementById('attachment-preview-bar');
    const filenameEl = document.getElementById('preview-filename');
    const filesizeEl = document.getElementById('preview-filesize');
    const removeBtn = document.getElementById('remove-attachment-btn');
    const statusEl = document.getElementById('upload-status');

    const urlInput = document.getElementById('chat-file-url');
    const nameInput = document.getElementById('chat-file-name');
    const typeInput = document.getElementById('chat-file-type');
    const sizeInput = document.getElementById('chat-file-size');

    if (!attachBtn || !fileInput) return;

    attachBtn.addEventListener('click', function () { fileInput.click(); });

    fileInput.addEventListener('change', function () {
      if (!fileInput.files || fileInput.files.length === 0) return;
      const file = fileInput.files[0];

      if (previewBar) previewBar.style.display = 'flex';
      if (statusEl) statusEl.textContent = 'Uploading...';
      if (filenameEl) filenameEl.textContent = file.name;
      if (filesizeEl) filesizeEl.textContent = '(' + (file.size / 1024).toFixed(1) + ' KB)';

      const formData = new FormData();
      formData.append('file', file);

      fetch('/api/hubs/upload', { method: 'POST', body: formData })
      .then(function (res) {
        if (!res.ok) return res.json().then(function (d) { throw new Error(d.message || 'Upload failed'); });
        return res.json();
      })
      .then(function (data) {
        if (urlInput) urlInput.value = data.file_url;
        if (nameInput) nameInput.value = data.file_name;
        if (typeInput) typeInput.value = data.file_type;
        if (sizeInput) sizeInput.value = data.file_size;
        if (statusEl) statusEl.textContent = '';
      })
      .catch(function (err) {
        if (statusEl) statusEl.textContent = 'Error: ' + err.message;
      });
    });

    function resetAttachment() {
      if (fileInput) fileInput.value = '';
      if (urlInput) urlInput.value = '';
      if (nameInput) nameInput.value = '';
      if (typeInput) typeInput.value = '';
      if (sizeInput) sizeInput.value = '';
      if (previewBar) previewBar.style.display = 'none';
      if (statusEl) statusEl.textContent = '';
    }

    if (removeBtn) removeBtn.addEventListener('click', resetAttachment);
    document.addEventListener('htmx:wsAfterSend', resetAttachment);
  }

  window.addEventListener('DOMContentLoaded', initAttachmentPipeline);

  // 7. Global minimalist keyboard shortcuts
  window.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') {
      if (document.activeElement && document.activeElement.blur) document.activeElement.blur();
    }
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
      e.preventDefault();
      const input = document.getElementById('chat-input-field');
      if (input) input.focus();
    }
  });
})();
