const CACHE_NAME = 'expense-tracker-v1';
const ASSETS = [
    '/',
    '/css/style.css',
    '/js/app.js',
    '/js/translations.js',
    '/js/modules/api.js',
    '/js/modules/state.js',
    '/js/modules/ui.js',
    '/js/modules/charts.js',
    '/js/modules/utils.js',
    'https://cdn.jsdelivr.net/npm/chart.js',
    'https://cdnjs.cloudflare.com/ajax/libs/qrcodejs/1.0.0/qrcode.min.js',
    'https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap'
];

self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(CACHE_NAME).then((cache) => {
            return cache.addAll(ASSETS);
        })
    );
});

self.addEventListener('fetch', (event) => {
    const url = new URL(event.request.url);

    // 1. Skip caching for API calls, Analytics, or non-GET requests
    if (event.request.method !== 'GET' || url.pathname.startsWith('/api/') || url.hostname.includes('api.exchangerate-api.com')) {
        return;
    }

    // 2. Stale-While-Revalidate strategy for static assets
    event.respondWith(
        caches.open(CACHE_NAME).then((cache) => {
            return cache.match(event.request).then((cachedResponse) => {
                const fetchPromise = fetch(event.request).then((networkResponse) => {
                    // Update cache with new version from network
                    if (networkResponse && networkResponse.status === 200) {
                        cache.put(event.request, networkResponse.clone());
                    }
                    return networkResponse;
                }).catch(() => {
                    // Fail silently, we'll try to use cache
                });

                // Return cached version immediately if available, else wait for network
                return cachedResponse || fetchPromise;
            });
        })
    );
});
