// YooMates Dumb Pipe JS - Zero Framework, Zero State (<50 lines)
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

  // Global minimalist keyboard shortcuts
  window.addEventListener('keydown', function (e) {
    // Escape blurs active input
    if (e.key === 'Escape') {
      if (document.activeElement && document.activeElement.blur) {
        document.activeElement.blur();
      }
    }
    // Ctrl+K or Cmd+K focuses channel switcher / input
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
      e.preventDefault();
      const input = document.getElementById('chat-input-field');
      if (input) input.focus();
    }
  });
})();
