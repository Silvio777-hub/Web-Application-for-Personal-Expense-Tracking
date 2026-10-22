// api.js
// Handles all network requests to the backend.

export async function apiFetch(url, options = {}) {
    const token = localStorage.getItem('token');
    const csrfToken = getCookie('csrf_token');

    const headers = {
        ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
        ...(csrfToken ? { 'X-CSRF-Token': csrfToken } : {})
    };

    // Auto-detect JSON vs multipart
    if (!(options.body instanceof FormData)) {
        headers['Content-Type'] = 'application/json';
    }

    const response = await fetch(url, {
        ...options,
        headers: { ...headers, ...(options.headers || {}) }
    });

    // Handle session expiration
    if (response.status === 401 && !url.includes('/api/auth/login')) {
        // If it's a background fetch or we specifically want to skip the global handler
        if (options.skipGlobalErrorHandler) {
            return null;
        }

        // Only redirect/clear if we think we are logged in
        if (localStorage.getItem('user')) {
            console.warn('Session expired. Redirecting to login.');
            localStorage.clear();
            window.dispatchEvent(new CustomEvent('unauthorized'));
        }
        return null;
    }

    if (!response.ok) {
        const err = await response.text();
        throw new Error(err);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
        const data = await response.json();
        return options.full ? { data, response } : data;
    }

    return options.full ? { data: null, response } : null;
}

export async function getExchangeRates(base = 'USD') {
    try {
        const data = await apiFetch(`/api/exchange-rates/${base}`);
        return data.rates;
    } catch (err) {
        console.error('Currency API Error:', err);
        return null;
    }
}

function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
}

