// YooMates Dumb Pipe JS - Zero Framework, Zero State (<170 lines)
// Strictly an event, offline cache, WebAssembly search, and DOM transport layer.

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

  // 7. WebAssembly In-Memory Search Engine
  function initWasmSearch() {
    if (typeof Go !== 'undefined' && 'WebAssembly' in window) {
      const go = new Go();
      WebAssembly.instantiateStreaming(fetch('/static/wasm/search.wasm'), go.importObject)
        .then(function (result) {
          go.run(result.instance);
          const badge = document.getElementById('wasm-badge');
          if (badge) badge.style.opacity = '1';
        })
        .catch(function (err) {
          console.warn('[wasm-search] Falling back to text matching:', err);
        });
    }

    const searchInput = document.getElementById('channel-search-input');
    const searchCount = document.getElementById('search-count');
    if (!searchInput) return;

    searchInput.addEventListener('input', function () {
      const query = searchInput.value.trim();
      const container = document.getElementById('messages-container');
      if (!container) return;

      const items = container.querySelectorAll('.message-item');
      if (query === '') {
        items.forEach(function (el) {
          el.classList.remove('msg-search-hidden', 'msg-search-match');
        });
        if (searchCount) searchCount.style.display = 'none';
        return;
      }

      const msgs = [];
      items.forEach(function (el) {
        const id = el.id.replace('msg-', '');
        const author = el.querySelector('.msg-author') ? el.querySelector('.msg-author').textContent.trim() : '';
        const body = el.querySelector('.msg-body') ? el.querySelector('.msg-body').textContent.trim() : '';
        const file = el.querySelector('.attachment-name') ? el.querySelector('.attachment-name').textContent.trim() : '';
        msgs.push({ id: id, author_name: author, content: body, file_name: file });
      });

      const matchIds = new Set();
      if (window.YooSearchWASM) {
        try {
          const raw = window.YooSearchWASM(query, JSON.stringify(msgs));
          const parsed = JSON.parse(raw);
          if (parsed && parsed.results) {
            parsed.results.forEach(function (r) { matchIds.add(r.id); });
          }
        } catch (e) {
          console.warn('[wasm-search] error:', e);
        }
      } else {
        const q = query.toLowerCase();
        msgs.forEach(function (m) {
          if (m.content.toLowerCase().includes(q) || m.author_name.toLowerCase().includes(q) || m.file_name.toLowerCase().includes(q)) {
            matchIds.add(m.id);
          }
        });
      }

      items.forEach(function (el) {
        const id = el.id.replace('msg-', '');
        if (matchIds.has(id)) {
          el.classList.remove('msg-search-hidden');
          el.classList.add('msg-search-match');
        } else {
          el.classList.add('msg-search-hidden');
          el.classList.remove('msg-search-match');
        }
      });

      if (searchCount) {
        searchCount.textContent = matchIds.size + ' matches';
        searchCount.style.display = 'inline';
      }
    });

    searchInput.addEventListener('keydown', function (e) {
      if (e.key === 'Escape') {
        searchInput.value = '';
        searchInput.dispatchEvent(new Event('input'));
        searchInput.blur();
      }
    });
  }

  window.addEventListener('DOMContentLoaded', initWasmSearch);

  // 8. Global minimalist keyboard shortcuts
  window.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') {
      if (document.activeElement && document.activeElement.blur) document.activeElement.blur();
    }
    if ((e.ctrlKey || e.metaKey) && (e.key === 'f' || e.key === 'F')) {
      const searchInput = document.getElementById('channel-search-input');
      if (searchInput) {
        e.preventDefault();
        searchInput.focus();
        searchInput.select();
      }
    }
    if ((e.ctrlKey || e.metaKey) && (e.key === 'k' || e.key === 'K')) {
      e.preventDefault();
      const input = document.getElementById('chat-input-field');
      if (input) input.focus();
    }
  });
})();
