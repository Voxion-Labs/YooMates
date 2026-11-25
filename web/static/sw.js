// YooMates Offline Resilience Service Worker (Engineered Minimalism)
const CACHE_NAME = 'yoomates-v2';

const STATIC_ASSETS = [
  '/',
  '/hubs',
  '/static/css/app.css',
  '/static/js/htmx.min.js',
  '/static/js/htmx-ws.js',
  '/static/js/wasm_exec.js',
  '/static/js/idb-cache.js',
  '/static/js/dumb-pipe.js',
  '/static/wasm/search.wasm'
];

// Install: pre-cache static assets
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(STATIC_ASSETS);
    }).then(() => self.skipWaiting())
  );
});

// Activate: purge stale caches and claim clients
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) => {
      return Promise.all(
        keys.map((key) => {
          if (key !== CACHE_NAME) {
            return caches.delete(key);
          }
        })
      );
    }).then(() => self.clients.claim())
  );
});

// Fetch: Stale-while-revalidate for static assets, network-first with cache fallback for pages
self.addEventListener('fetch', (event) => {
  const req = event.request;
  const url = new URL(req.url);

  // Bypass non-GET, WebSockets, uploads, and dynamic API endpoints
  if (req.method !== 'GET' || url.pathname.startsWith('/ws') || url.pathname.startsWith('/api/')) {
    return;
  }

  // 1. Static Assets: Stale-While-Revalidate (0ms instant return from cache)
  if (url.pathname.startsWith('/static/')) {
    event.respondWith(
      caches.match(req).then((cachedResponse) => {
        const fetchPromise = fetch(req).then((networkResponse) => {
          if (networkResponse && networkResponse.status === 200) {
            const responseClone = networkResponse.clone();
            caches.open(CACHE_NAME).then((cache) => cache.put(req, responseClone));
          }
          return networkResponse;
        }).catch(() => cachedResponse);

        return cachedResponse || fetchPromise;
      })
    );
    return;
  }

  // 2. Navigation / Hubs Page: Network-First with Offline Cache Fallback
  if (req.mode === 'navigate' || url.pathname === '/' || url.pathname === '/hubs') {
    event.respondWith(
      fetch(req).then((networkResponse) => {
        if (networkResponse && networkResponse.status === 200) {
          const responseClone = networkResponse.clone();
          caches.open(CACHE_NAME).then((cache) => cache.put(req, responseClone));
        }
        return networkResponse;
      }).catch(() => {
        return caches.match('/hubs').then((cachedPage) => {
          return cachedPage || caches.match('/');
        });
      })
    );
    return;
  }

  // Default fetch
  event.respondWith(
    caches.match(req).then((cached) => cached || fetch(req))
  );
});
