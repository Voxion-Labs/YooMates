// YooMates Dumb Pipe JS - Zero Framework, Zero State (<90 lines)
// Strictly an event and DOM transport layer.

(function () {
  'use strict';

  function scrollToBottom() {
    const stream = document.getElementById('message-stream');
    if (stream) {
      stream.scrollTop = stream.scrollHeight;
    }
  }

  // Scroll to bottom on load
  window.addEventListener('DOMContentLoaded', scrollToBottom);

  // Auto-scroll on HTMX DOM insertions or WebSocket swaps
  document.addEventListener('htmx:afterSwap', function (evt) {
    if (evt.target && (evt.target.id === 'message-stream' || evt.target.closest('#message-stream'))) {
      scrollToBottom();
    }
  });

  // Clear chat input after HTMX WebSocket sends message
  document.addEventListener('htmx:wsAfterSend', function () {
    const input = document.getElementById('chat-input-field');
    if (input) {
      input.value = '';
      input.focus();
    }
  });

  // Attachment upload and preview handling
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

    attachBtn.addEventListener('click', function () {
      fileInput.click();
    });

    fileInput.addEventListener('change', function () {
      if (!fileInput.files || fileInput.files.length === 0) return;
      const file = fileInput.files[0];

      if (previewBar) previewBar.style.display = 'flex';
      if (statusEl) statusEl.textContent = 'Uploading...';
      if (filenameEl) filenameEl.textContent = file.name;
      if (filesizeEl) filesizeEl.textContent = '(' + (file.size / 1024).toFixed(1) + ' KB)';

      const formData = new FormData();
      formData.append('file', file);

      fetch('/api/hubs/upload', {
        method: 'POST',
        body: formData
      })
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

    if (removeBtn) {
      removeBtn.addEventListener('click', resetAttachment);
    }

    document.addEventListener('htmx:wsAfterSend', resetAttachment);
  }

  window.addEventListener('DOMContentLoaded', initAttachmentPipeline);

  // Global minimalist keyboard shortcuts
  window.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') {
      if (document.activeElement && document.activeElement.blur) {
        document.activeElement.blur();
      }
    }
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
      e.preventDefault();
      const input = document.getElementById('chat-input-field');
      if (input) input.focus();
    }
  });
})();
