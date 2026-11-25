// YooMates IndexedDB Local Storage Cache (<80 lines)
// Instant 0ms offline chat load with schema parity to MySQL

(function () {
  'use strict';

  const DB_NAME = 'yoomates_client_db';
  const DB_VERSION = 1;

  function openDB() {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(DB_NAME, DB_VERSION);

      request.onupgradeneeded = (e) => {
        const db = e.target.result;
        if (!db.objectStoreNames.contains('messages')) {
          const msgStore = db.createObjectStore('messages', { keyPath: 'id' });
          msgStore.createIndex('channel_id', 'channel_id', { unique: false });
          msgStore.createIndex('created_at', 'created_at', { unique: false });
        }
      };

      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });
  }

  // Save a message into IndexedDB
  function saveMessage(msg) {
    if (!msg || !msg.id) return Promise.resolve();
    return openDB().then((db) => {
      return new Promise((resolve, reject) => {
        const tx = db.transaction('messages', 'readwrite');
        const store = tx.objectStore('messages');
        store.put(msg);
        tx.oncomplete = () => resolve();
        tx.onerror = () => reject(tx.error);
      });
    }).catch((err) => console.warn('[idb-cache] Save error:', err));
  }

  // Get all cached messages for a channel
  function getChannelMessages(channelId) {
    return openDB().then((db) => {
      return new Promise((resolve, reject) => {
        const tx = db.transaction('messages', 'readonly');
        const store = tx.objectStore('messages');
        const index = store.index('channel_id');
        const request = index.getAll(channelId);

        request.onsuccess = () => resolve(request.result || []);
        request.onerror = () => reject(request.error);
      });
    }).catch(() => []);
  }

  // Scrape and cache all messages currently in the DOM
  function syncDOMToCache(channelId) {
    if (!channelId) return;
    const container = document.getElementById('messages-container');
    if (!container) return;

    const items = container.querySelectorAll('.message-item');
    items.forEach((item) => {
      const id = item.id.replace('msg-', '');
      if (!id) return;

      const authorEl = item.querySelector('.msg-author');
      const avatarEl = item.querySelector('.msg-avatar');
      const timeEl = item.querySelector('.msg-time');
      const bodyEl = item.querySelector('.msg-body');
      const imgEl = item.querySelector('.msg-attachment-img');
      const fileCardEl = item.querySelector('.msg-attachment-card');

      let fileUrl = '';
      let fileName = '';
      if (imgEl) {
        fileUrl = imgEl.getAttribute('src') || '';
        fileName = imgEl.getAttribute('alt') || '';
      } else if (fileCardEl) {
        fileUrl = fileCardEl.getAttribute('href') || '';
        const nameEl = fileCardEl.querySelector('.attachment-name');
        fileName = nameEl ? nameEl.textContent.trim() : '';
      }

      saveMessage({
        id: id,
        channel_id: channelId,
        author_name: authorEl ? authorEl.textContent.trim() : 'Member',
        initials: avatarEl ? avatarEl.textContent.trim() : 'U',
        time: timeEl ? timeEl.textContent.trim() : '',
        content: bodyEl ? bodyEl.textContent.trim() : '',
        file_url: fileUrl,
        file_name: fileName,
        created_at: Date.now()
      });
    });
  }

  window.YooCache = {
    openDB: openDB,
    saveMessage: saveMessage,
    getChannelMessages: getChannelMessages,
    syncDOMToCache: syncDOMToCache
  };
})();
