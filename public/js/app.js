// app.js - Entry Point
// Coordinates all modules and sets up event listeners.

import { state, elements, initElements } from './modules/state.js';
import { apiFetch, getExchangeRates } from './modules/api.js';
import {
    setupTheme,
    translateUI,
    renderExpensesList,
    renderTopLowCategories,
    updateGoalStatus,
    getCurrencySymbol,
    formatDate,
    t,
    escapeHTML
} from './modules/ui.js';
import { initCharts } from './modules/charts.js';

console.log('App initialization starting...');
window.onerror = (msg, url, line) => {
    console.error(`JS ERROR: ${msg} at ${line}`);
};

// UI Helpers for safe DOM access
const val = (id) => {
    const el = document.getElementById(id);
    if (!el) {
        console.warn(`Element #${id} not found`);
        return '';
    }
    return el.value || '';
};
const setVal = (id, v) => {
    const el = document.getElementById(id);
    if (el) el.value = v;
};
const text = (id, t) => {
    const el = document.getElementById(id);
    if (el) el.innerText = t;
};
const inner = (id, h) => {
    const el = document.getElementById(id);
    if (el) el.innerHTML = h;
};

const saveUser = (user) => {
    if (!user) {
        localStorage.removeItem('user');
        return;
    }
    // Filter out potential PII or sensitive internal fields
    const sanitized = {
        id: user.id,
        username: user.username,
        role: user.role,
        currency: user.currency,
        language: user.language,
        date_format: user.date_format,
        profile_picture_url: user.profile_picture_url,
        monthly_savings_goal: user.monthly_savings_goal,
        notify_enabled: user.notify_enabled
    };
    state.user = sanitized;
    localStorage.setItem('user', JSON.stringify(sanitized));
    if (user.is_read_only !== undefined) {
        localStorage.setItem('is_read_only', user.is_read_only);
        state.isReadOnly = user.is_read_only;
    }
};

const LOO_LIMIT = 5;
const LOOP_WINDOW = 10000; // 10 seconds

function checkRedirectLoop() {
    const now = Date.now();
    let history = JSON.parse(sessionStorage.getItem('reload_history') || '[]');
    history = history.filter(t => now - t < LOOP_WINDOW);
    history.push(now);
    sessionStorage.setItem('reload_history', JSON.stringify(history));

    if (history.length > LOO_LIMIT) {
        console.error('REDIRECT LOOP DETECTED. HALTING APPLICATION.');
        // Show a huge error message on screen
        document.body.innerHTML = `
            <div style="background: #c0392b; color: white; padding: 2rem; text-align: center; font-family: sans-serif; height: 100vh; display: flex; flex-direction: column; justify-content: center;">
                <h1>⚠️ Security Halt</h1>
                <p>An infinite reload loop was detected and blocked for your safety.</p>
                <p>Please clear your browser cache and cookies, then <button onclick="sessionStorage.clear(); location.reload();" style="padding: 10px 20px; border: none; border-radius: 5px; cursor: pointer; font-weight: bold;">Try Again</button></p>
            </div>
        `;
        throw new Error('Reload loop halted');
    }
}

function safeReload() {
    checkRedirectLoop();
    window.location.reload();
}

// Toggle password visibility
window.togglePasswordVisibility = (inputId) => {
    const input = document.getElementById(inputId);
    const btn = input.nextElementSibling;
    const type = input.getAttribute('type') === 'password' ? 'text' : 'password';
    input.setAttribute('type', type);

    // Toggle SVG icon paths (basic switch)
    if (type === 'text') {
        btn.innerHTML = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 19c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path><line x1="1" y1="1" x2="23" y2="23"></line></svg>`;
    } else {
        btn.innerHTML = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>`;
    }
};

let initInProgress = false;

window.addEventListener('unauthorized', () => {
    console.error('Unauthorized event received. Cleaning up.');
    localStorage.clear();
    state.user = null;
    // Clear cookies explicitly if possible (though HttpOnly prevents JS access)
    document.cookie = 'auth_token=; Max-Age=-99999999;';
    showView('auth');
});

async function initApp() {
    if (initInProgress) return;
    initInProgress = true;

    initElements(); // MUST BE FIRST
    setupTheme();
    setupEventListeners();

    // Check for Recovery Token in URL
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token');

    if (token) {
        console.log('Recovery token detected. Opening reset UI...');
        localStorage.clear(); // Clear any stale sessions
        showView('auth');
        elements.authSection.classList.add('hidden');
        elements.resetModal.classList.remove('hidden');
        document.getElementById('reset-step-1').classList.add('hidden');
        document.getElementById('reset-step-2').classList.remove('hidden');
        const tokenInput = document.getElementById('reset-token-hidden');
        if (tokenInput) tokenInput.value = token;
        return; // Halt normal init to focus on reset
    }

    initAuth(); // Initialize Remember Me
    if (state.user && state.user.id && localStorage.getItem('user')) {
        showView(state.user.role === 'admin' ? 'admin' : 'dashboard');

        // Handle Impersonation UI
        if (localStorage.getItem('admin_token')) {
            document.getElementById('impersonation-banner')?.classList.remove('hidden');
        }

        try {
            await refreshData();
            if (state.user && state.user.role === 'admin') {
                await refreshAdminData();
            }
        } catch (e) {
            console.error('Init data Load failed:', e);
        }

        translateUI(state.user?.language || 'en');
    } else {
        localStorage.clear();
        showView('auth');
    }

    // Diagnostic Probe
    const jsDiag = document.getElementById('diag-js');
    if (jsDiag) {
        jsDiag.innerText = 'OK';
        jsDiag.style.color = '#27ae60';
    }

    try {
        await apiFetch('/api/categories', { skipGlobalErrorHandler: true });
        const apiDiag = document.getElementById('diag-api');
        if (apiDiag) {
            apiDiag.innerText = 'OK';
            apiDiag.style.color = '#27ae60';
        }
    } catch (e) { }
}

window.onload = () => {
    console.log('DOM and Libraries loaded. Initializing...');
    initApp().catch(err => alert(`INIT FATAL: ${err.message}`));
};

// PWA Install Prompt handling
let deferredPrompt;
window.addEventListener('beforeinstallprompt', (e) => {
    // Prevent the mini-infobar from appearing on mobile
    e.preventDefault();
    // Stash the event so it can be triggered later.
    deferredPrompt = e;
    console.log('PWA Install prompt stashed.');

    // Optional: show a custom install button in the UI
    const installBtn = document.getElementById('pwa-install-btn');
    if (installBtn) {
        installBtn.classList.remove('hidden');
        installBtn.onclick = async () => {
            if (!deferredPrompt) return;
            deferredPrompt.prompt();
            const { outcome } = await deferredPrompt.userChoice;
            console.log(`User responded to install prompt: ${outcome}`);
            deferredPrompt = null;
            installBtn.classList.add('hidden');
        };
    }
});

if (document.readyState === 'complete' || document.readyState === 'interactive') {
    console.log('DOM ready detected in module. Triggering initApp.');
    initApp().catch(err => console.error('Immediate init failed:', err));
}

function showView(view) {
    state.currentView = view;
    // Force close all modals to ensure clean navigation
    document.querySelectorAll('.modal').forEach(m => m.classList.add('hidden'));

    // Direct DOM access for reliability
    const sections = ['auth-section', 'dashboard-section', 'admin-section'];
    sections.forEach(id => {
        const el = document.getElementById(id);
        if (el) el.classList.add('hidden');
    });

    // Toggle Nav Elements
    const userNav = document.getElementById('user-nav');
    const navCenter = document.querySelector('.nav-center');

    if (view === 'auth') {
        if (userNav) userNav.classList.add('hidden');
        if (navCenter) navCenter.classList.add('hidden');
        document.getElementById('auth-section').classList.remove('hidden');
    } else {
        if (userNav) userNav.classList.remove('hidden');
        if (navCenter) navCenter.classList.remove('hidden');

        const usernameDisplay = document.getElementById('username-display');
        if (usernameDisplay) usernameDisplay.innerText = state.user.username;

        // Profile Pic logic
        const navAvatar = document.getElementById('nav-avatar');
        const navPlaceholder = document.getElementById('nav-avatar-placeholder');

        if (state.user.profile_picture_url && navAvatar) {
            navAvatar.src = state.user.profile_picture_url;
            navAvatar.classList.remove('hidden');
            if (navPlaceholder) navPlaceholder.classList.add('hidden');
        } else if (navAvatar) {
            navAvatar.classList.add('hidden');
            if (navPlaceholder) {
                navPlaceholder.classList.remove('hidden');
                navPlaceholder.innerText = state.user.username[0].toUpperCase();
            }
        }

        if (state.user.role === 'admin') {
            const adminBtn = document.getElementById('admin-btn');
            const dashboardBtn = document.getElementById('dashboard-btn');
            if (adminBtn) adminBtn.classList.remove('hidden');
            if (dashboardBtn) dashboardBtn.classList.remove('hidden');
        }

        // Toggle Impersonation Banner
        const impBanner = document.getElementById('impersonation-banner');
        if (state.isReadOnly && impBanner) {
            impBanner.classList.remove('hidden');
        } else if (impBanner) {
            impBanner.classList.add('hidden');
        }

        const target = document.getElementById(view + '-section');
        if (target) target.classList.remove('hidden');
    }
}

function setupEventListeners() {
    console.log('Finalizing event listeners...');

    // Auth - HIGHEST PRIORITY
    const authForm = document.getElementById('auth-form');
    if (authForm) {
        authForm.onsubmit = handleAuth;
        console.log('Auth form intercepted.');
    }

    const logoutBtn = document.getElementById('logout-btn');
    if (logoutBtn) logoutBtn.onclick = handleLogout;

    const forgotLink = document.getElementById('forgot-password-link');
    if (forgotLink) {
        forgotLink.onclick = (e) => {
            e.preventDefault();
            console.log('Forgot password link clicked. Forcing UI...');
            // Direct DOM access for high reliability
            const authSect = document.getElementById('auth-section');
            const resetMod = document.getElementById('reset-modal');
            if (authSect) authSect.classList.add('hidden');
            if (resetMod) resetMod.classList.remove('hidden');
        };
    }

    if (elements.resetRequestForm) elements.resetRequestForm.onsubmit = handleResetRequest;
    if (elements.resetVerifyForm) elements.resetVerifyForm.onsubmit = handleResetVerify;

    // Expenses
    if (elements.addExpenseBtn) elements.addExpenseBtn.onclick = () => openExpenseModal();
    if (elements.navAddExpenseBtn) elements.navAddExpenseBtn.onclick = () => openExpenseModal();
    if (elements.expenseForm) elements.expenseForm.onsubmit = handleAddExpense;
    if (elements.closeModal) elements.closeModal.onclick = () => elements.expenseModal.classList.add('hidden');

    const receiptInput = document.getElementById('exp-receipt');
    if (receiptInput) {
        receiptInput.onchange = handleReceiptUpload;
    }


    // Dashboard Filters
    if (elements.expenseFilter) {
        elements.expenseFilter.onchange = (e) => {
            if (e.target.value === 'custom') {
                elements.customDatePicker.classList.remove('hidden');
            } else {
                elements.customDatePicker.classList.add('hidden');
                refreshData();
            }
        };
    }
    if (elements.applyFilterBtn) elements.applyFilterBtn.onclick = refreshData;

    // Currency Converter
    const convBtn = document.getElementById('convert-btn');
    if (convBtn) convBtn.onclick = handleCurrencyConversion;

    // Admin Controls
    const maintToggle = document.getElementById('maintenance-toggle');
    if (maintToggle) {
        maintToggle.onchange = (e) => {
            window.updateConfig('maintenance_mode', e.target.checked ? 'true' : 'false');
        };
    }

    const saveAnnBtn = document.getElementById('save-announcement-btn');
    if (saveAnnBtn) {
        saveAnnBtn.onclick = () => {
            const msg = val('global-announcement-input');
            window.updateConfig('global_announcement', msg);
        };
    }

    const exportAllBtn = document.getElementById('admin-export-all-btn');
    if (exportAllBtn) {
        exportAllBtn.onclick = () => {
            window.location.href = '/api/expenses/export?all=true';
        };
    }

    const backupBtn = document.getElementById('admin-create-backup-btn');
    if (backupBtn) {
        backupBtn.onclick = async () => {
            const statusEl = document.getElementById('backup-status');
            if (statusEl) statusEl.innerText = 'Creating backup...';
            try {
                const data = await apiFetch('/api/admin/system/backup', { method: 'POST' });
                showToast(data.message, 'success');
                if (statusEl) statusEl.innerText = `Last backup: ${data.filename}`;
            } catch (err) {
                showToast('Backup failed: ' + err.message, 'error');
                if (statusEl) statusEl.innerText = 'Last backup failed.';
            }
        };
    }

    const clearLogsBtn = document.getElementById('admin-clear-logs-btn');
    if (clearLogsBtn) {
        clearLogsBtn.onclick = async () => {
            if (!confirm('Are you absolutely sure? This will PERMANENTLY delete all activity logs.')) return;
            try {
                await apiFetch('/api/admin/audit-logs', { method: 'DELETE' });
                showToast('All activity logs cleared!', 'success');
                refreshAdminData();
            } catch (err) {
                showToast('Failed to clear logs: ' + err.message, 'error');
            }
        };
    }

    const clearExpensesBtn = document.getElementById('admin-clear-transactions-btn');
    if (clearExpensesBtn) {
        clearExpensesBtn.onclick = async () => {
            if (!confirm('WARNING: This will PERMANENTLY delete all transactions on the entire platform. This cannot be undone. Proceed?')) return;
            try {
                await apiFetch('/api/admin/expenses/all', { method: 'DELETE' });
                showToast('All platform expenses cleared!', 'success');
                refreshData();
                refreshAdminData();
            } catch (err) {
                showToast('Failed to clear expenses: ' + err.message, 'error');
            }
        };
    }

    // Theme & Settings
    const dashboardBtn = document.getElementById('dashboard-btn');
    if (dashboardBtn) {
        dashboardBtn.onclick = () => {
            showView('dashboard');
            refreshData();
        };
    }

    const adminBtn = document.getElementById('admin-btn');
    if (adminBtn) {
        adminBtn.onclick = () => {
            showView('admin');
            refreshData(); // Triggers logic inside refreshData to load admin stats if view is admin
        };
    }

    if (elements.settingsBtn) {
        elements.settingsBtn.onclick = () => {
            elements.settingsModal.classList.remove('hidden');
            populateSettings();
        };
    }

    // Tab Switching Logic
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.onclick = () => {
            document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            document.querySelectorAll('.tab-pane').forEach(pane => pane.classList.add('hidden'));
            const targetId = btn.getAttribute('data-tab');
            const targetPane = document.getElementById(targetId);
            if (targetPane) targetPane.classList.remove('hidden');
        };
    });

    // Settings Forms
    if (elements.preferencesForm) elements.preferencesForm.onsubmit = handlePreferencesUpdate;
    if (elements.settingsForm) elements.settingsForm.onsubmit = handleProfileUpdate;
    if (elements.passwordForm) elements.passwordForm.onsubmit = handlePasswordUpdate;



    // Search & Amount Filters (Debounced)
    const debouncedRefresh = debounce(refreshData, 500);
    elements.expenseSearch?.addEventListener('input', debouncedRefresh);
    elements.filterMinAmount?.addEventListener('input', debouncedRefresh);
    elements.filterMaxAmount?.addEventListener('input', debouncedRefresh);

    // Export & Trash
    elements.exportBtn?.addEventListener('click', handleExport);
    elements.viewTrashBtn?.addEventListener('click', handleViewTrash);

    // Admin Register
    elements.adminRegisterForm?.addEventListener('submit', handleAdminRegister);

    // Bulk Selection
    elements.selectAllExpenses?.addEventListener('change', handleSelectAll);
    elements.expensesList?.addEventListener('change', (e) => {
        if (e.target.classList.contains('expense-checkbox')) {
            updateBulkBar();
        }
    });

    // Global Close Modal
    window.closeModal = (id) => {
        const modal = document.getElementById(id);
        if (modal) modal.classList.add('hidden');
    };
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

function populateSettings() {
    if (!state.user) return;

    // Preferences
    setVal('set-currency', state.user.currency || 'USD');
    setVal('set-language', state.user.language || 'en');
    setVal('set-date-format', state.user.date_format || 'YYYY-MM-DD');
    const setNotify = document.getElementById('set-notify');
    if (setNotify) setNotify.checked = state.user.notify_enabled;
    setVal('set-monthly-savings-goal', state.user.monthly_savings_goal || 0);
    setVal('set-email', state.user.email || '');
}

async function handlePreferencesUpdate(e) {
    e.preventDefault();
    const prefs = {
        currency: val('set-currency'),
        language: val('set-language'),
        date_format: val('set-date-format'),
        notify_enabled: document.getElementById('set-notify').checked,
        monthly_savings_goal: parseFloat(val('set-monthly-savings-goal')) || 0
    };

    try {
        const updatedUser = await apiFetch('/api/user/preferences', {
            method: 'PUT',
            body: JSON.stringify(prefs)
        });
        showToast(t('prefs_update_success'), 'success');
        saveUser(updatedUser);
        translateUI(state.user.language);
        refreshData();
        window.closeModal('settings-modal');
    } catch (err) {
        showToast(t('update_failed') + err.message, 'error');
    }
}

async function handleProfileUpdate(e) {
    e.preventDefault();
    const email = val('set-email');
    const fileInput = document.getElementById('profile-pic-input');
    let profilePicUrl = state.user.profile_picture_url;

    try {
        if (fileInput.files.length > 0) {
            const formData = new FormData();
            formData.append('profile_pic', fileInput.files[0]);
            // Use secure wrapper for CSRF + Auth
            const data = await apiFetch('/api/user/upload-pic', {
                method: 'POST',
                body: formData
            });
            profilePicUrl = data.url;
        }

        const updatedUser = await apiFetch('/api/user/profile', {
            method: 'PUT',
            body: JSON.stringify({ email, profile_picture_url: profilePicUrl })
        });

        saveUser(updatedUser);

        // Live UI Update
        const navAvatar = document.getElementById('nav-avatar');
        if (navAvatar && profilePicUrl) {
            navAvatar.src = profilePicUrl;
            navAvatar.classList.remove('hidden');
            const placeholder = document.getElementById('nav-avatar-placeholder');
            if (placeholder) placeholder.classList.add('hidden');
        }

        const usernameDisplay = document.getElementById('username-display');
        if (usernameDisplay) usernameDisplay.innerText = updatedUser.username;

        showToast(t('profile_update_success'), 'success');
        refreshData();
    } catch (err) {
        showToast(t('profile_update_failed') + err.message, 'error');
    }
}



async function handlePasswordUpdate(e) {
    e.preventDefault();
    const current = val('current-pass');
    const newPass = val('new-pass');
    const confirmPass = val('confirm-pass');

    if (newPass !== confirmPass) {
        alert(t('password_match_error'));
        return;
    }

    try {
        await apiFetch('/api/user/change-password', {
            method: 'POST',
            body: JSON.stringify({ current, new: newPass })
        });
        showToast(t('password_update_success'), 'success');
        elements.passwordForm.reset();
    } catch (err) {
        showToast(t('password_update_failed') + err.message, 'error');
    }
}

function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
}

async function refreshData() {
    try {
        const filter = elements.expenseFilter?.value || '';
        const search = elements.expenseSearch?.value || '';
        const minAmt = elements.filterMinAmount?.value || '';
        const maxAmt = elements.filterMaxAmount?.value || '';

        const params = new URLSearchParams({
            filter: filter,
            q: search,
            min_amount: minAmt,
            max_amount: maxAmt,
            limit: state.pagination.limit,
            offset: 0
        });

        if (filter === 'custom' && elements.filterStartDate.value && elements.filterEndDate.value) {
            params.set('start_date', elements.filterStartDate.value);
            params.set('end_date', elements.filterEndDate.value);
        }

        if (state.user.role === 'admin') {
            params.set('all', 'true');
        }

        const url = `/api/expenses?${params.toString()}`;

        const [resExpenses, resCategories, resAll] = await Promise.all([
            apiFetch(url, { skipGlobalErrorHandler: true }).catch(() => []),
            apiFetch('/api/categories', { skipGlobalErrorHandler: true }).catch(() => []),
            apiFetch(state.user.role === 'admin' ? '/api/expenses?all=true' : '/api/expenses', { skipGlobalErrorHandler: true }).catch(() => [])
        ]);

        const expenses = resExpenses || [];
        const categories = resCategories || [];
        const all = resAll || [];

        state.expenses = expenses;
        state.categories = categories;
        state.pagination.page = 1;
        state.pagination.hasMore = expenses.length === state.pagination.limit;


        // UI Updates
        renderExpensesList(expenses);
        updateDashboardMetrics(all);
        try {
            initCharts(all);
        } catch (chartErr) {
            console.warn('Dashboard charts failed to render:', chartErr);
        }
    } catch (err) {
        console.error('Data refresh failed:', err);
        showToast(t('load_failed') || 'Failed to load dashboard data. Please reload.', 'error');
    }
}

async function loadMoreExpenses() {
    if (!state.pagination.hasMore) return;

    const btn = document.getElementById('load-more-btn');
    if (btn) btn.disabled = true;

    try {
        const offset = state.pagination.page * state.pagination.limit;
        const filter = elements.expenseFilter?.value || '';
        const search = elements.expenseSearch?.value || '';
        const minAmt = elements.filterMinAmount?.value || '';
        const maxAmt = elements.filterMaxAmount?.value || '';

        const params = new URLSearchParams({
            filter: filter,
            q: search,
            min_amount: minAmt,
            max_amount: maxAmt,
            limit: state.pagination.limit,
            offset: offset
        });

        if (filter === 'custom' && elements.filterStartDate.value && elements.filterEndDate.value) {
            params.set('start_date', elements.filterStartDate.value);
            params.set('end_date', elements.filterEndDate.value);
        }

        const data = await apiFetch(`/api/expenses?${params.toString()}`);
        if (data && data.length > 0) {
            state.expenses = [...state.expenses, ...data];
            state.pagination.page++;
            state.pagination.hasMore = data.length === state.pagination.limit;
            renderExpensesList(state.expenses);
        } else {
            state.pagination.hasMore = false;
            renderExpensesList(state.expenses);
        }
    } catch (err) {
        console.error('Load more failed:', err);
    } finally {
        if (btn) btn.disabled = false;
    }
}

window.loadMoreExpenses = loadMoreExpenses;


function updateDashboardMetrics(all) {
    const symbol = getCurrencySymbol(state.user.currency || 'USD');
    const total = all.reduce((sum, e) => sum + e.amount, 0);

    text('total-balance', `${symbol}${total.toFixed(2)}`);

    updateGoalStatus(all);
    renderTopLowCategories(all);
}

// Remember Me Logic
function initAuth() {
    const savedUser = localStorage.getItem('saved_username');
    if (savedUser) {
        const userInput = document.getElementById('username');
        const rememberBox = document.getElementById('remember-me');
        if (userInput) userInput.value = savedUser;
        if (rememberBox) rememberBox.checked = true;
    }
}

// Auth Handlers
async function handleAuth(e) {
    if (e) e.preventDefault();
    const btn = document.getElementById('auth-submit');
    if (btn) {
        btn.disabled = true;
        btn.innerText = 'Verifying...';
    }

    console.log('Login attempt started at:', new Date().toISOString());
    const username = val('username');
    const password = val('password');

    if (!username || !password) {
        alert('Please enter both username and password.');
        return;
    }
    try {
        const url = state.isLogin ? '/api/auth/login' : '/api/auth/register';
        const body = state.isLogin ? { username, password } : { username, password, email: val('email') };

        const data = await apiFetch(url, {
            method: 'POST',
            body: JSON.stringify(body)
        });

        // Remember Me Logic
        if (state.isLogin && document.getElementById('remember-me')) {
            if (document.getElementById('remember-me').checked) {
                localStorage.setItem('saved_username', username);
            } else {
                localStorage.removeItem('saved_username');
            }
        }



        if (data.token) {
            localStorage.setItem('token', data.token);
        }
        if (data.is_read_only !== undefined) {
            state.isReadOnly = data.is_read_only;
            localStorage.setItem('is_read_only', data.is_read_only);
        }
        saveUser(data.user);
        // Successful manual login: reset reload loop protection
        sessionStorage.removeItem('reload_history');
        safeReload();
    } catch (err) {
        alert('LOGIN FAILED: ' + err.message);
    } finally {
        if (btn) {
            btn.disabled = false;
            btn.innerText = state.isLogin ? 'Login' : 'Register';
        }
    }
}

async function handleLogout() {
    try {
        await apiFetch('/api/auth/logout', { method: 'POST' });
        localStorage.clear();
        safeReload();
    } catch (err) {
        console.error('Logout failed:', err);
        localStorage.clear();
        safeReload();
    }
}

// Expense Handlers
function openExpenseModal(id = null) {
    elements.expenseModal.classList.remove('hidden');
    const catSelect = document.getElementById('exp-category');
    // Deduplicate by name to handle potential DB migration overlapping
    const uniqueCategories = [];
    const seenNames = new Set();
    state.categories.forEach(c => {
        if (!seenNames.has(c.name)) {
            seenNames.add(c.name);
            uniqueCategories.push(c);
        }
    });
    catSelect.innerHTML = uniqueCategories.map(c => `<option value="${c.id}">${c.name}</option>`).join('');

    if (!id) {
        elements.expenseForm.reset();
        setVal('exp-id', '');
        window.clearReceipt();
    }
}

async function handleAddExpense(e) {
    e.preventDefault();
    const expense = {
        amount: parseFloat(val('exp-amount')),
        category_id: parseInt(val('exp-category')),
        description: val('exp-desc'),
        date: val('exp-date'),
        currency: state.user.currency || 'USD',
        receipt_url: val('exp-receipt-url')
    };

    const id = val('exp-id');
    let response;

    try {
        const { data, response } = await apiFetch('/api/expenses', {
            method: id ? 'PUT' : 'POST',
            body: JSON.stringify(id ? { ...expense, id: parseInt(id) } : expense),
            full: true
        });

        elements.expenseModal.classList.add('hidden');


        refreshData();
        if (state.user && state.user.role === 'admin') {
            refreshAdminData();
        }
    } catch (err) {
        showToast(t('save_failed') + err.message, 'error');
    }
}

async function handleReceiptUpload(e) {
    const file = e.target.files[0];
    if (!file) return;

    const statusEl = document.getElementById('receipt-status');
    const clearBtn = document.getElementById('clear-receipt');
    const urlInput = document.getElementById('exp-receipt-url');

    if (statusEl) statusEl.innerText = 'Uploading...';

    const formData = new FormData();
    formData.append('receipt', file);

    try {
        const data = await apiFetch('/api/expenses/upload-receipt', {
            method: 'POST',
            body: formData
        });

        if (urlInput) urlInput.value = data.url;
        if (statusEl) statusEl.innerText = 'Receipt uploaded: ' + file.name;
        if (clearBtn) clearBtn.classList.remove('hidden');

        // AI Auto-Fill Logic
        if (data.ai_data && data.ai_data.confidence > 0.5) {
            console.log('AI detected receipt data:', data.ai_data);
            if (data.ai_data.amount > 0) setVal('exp-amount', data.ai_data.amount);
            if (data.ai_data.description) setVal('exp-desc', data.ai_data.description);
            if (data.ai_data.date) {
                const date = data.ai_data.date.split('T')[0];
                setVal('exp-date', date);
            }
            showToast(t('ai_autofill_success') || 'Uploaded Successfully', 'info');
        }
    } catch (err) {
        if (statusEl) statusEl.innerText = 'Upload failed';
        alert('Receipt upload failed: ' + err.message);
    }
}

window.clearReceipt = () => {
    const input = document.getElementById('exp-receipt');
    const urlInput = document.getElementById('exp-receipt-url');
    const statusEl = document.getElementById('receipt-status');
    const clearBtn = document.getElementById('clear-receipt');

    if (input) input.value = '';
    if (urlInput) urlInput.value = '';
    if (statusEl) statusEl.innerText = t('click_upload');
    if (clearBtn) clearBtn.classList.add('hidden');
};

window.downloadReceipt = (url, filename = 'receipt') => {
    // Append download=true if not present
    const separator = url.includes('?') ? '&' : '?';
    const downloadUrl = url.includes('download=true') ? url : `${url}${separator}download=true`;

    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
};



async function handleResetRequest(e) {
    if (e) e.preventDefault();
    const email = val('reset-email-input');
    if (!email) {
        alert('Please enter your email address.');
        return;
    }

    const btn = e.target.querySelector('button[type="submit"]');
    if (btn) btn.disabled = true;

    try {
        await apiFetch('/api/auth/reset-request', {
            method: 'POST',
            body: JSON.stringify({ email })
        });

        // Instructional feedback
        showToast('Please check your Email Inbox or Spam to Create your New Password', 'success');

        // Close modal or show success state
        if (elements.resetModal) elements.resetModal.classList.add('hidden');
        showView('auth');
    } catch (err) {
        alert(err.message); // Keep alert for error so validation is seen clearly
    } finally {
        if (btn) btn.disabled = false;
    }
}

async function handleResetVerify(e) {
    e.preventDefault();
    const token = val('reset-token-hidden');
    const password = val('reset-new-pass');

    if (!token) {
        alert('Missing recovery token. Please click the link in your email again.');
        return;
    }

    try {
        await apiFetch('/api/auth/reset-verify', {
            method: 'POST',
            body: JSON.stringify({ token, password })
        });
        alert('Password reset successful! Please login with your new password.');
        window.location.href = '/'; // Clear token from URL and refresh
    } catch (err) {
        alert('Verification Failed: ' + err.message);
    }
}

// Global exposure for legacy HTML onclicks (temporary until full migration)
window.viewReceipt = (url) => window.open(url, '_blank');
window.editExpense = (id) => {
    const exp = state.expenses.find(e => e.id === id);
    if (!exp) return;
    openExpenseModal(id);
    setVal('exp-id', exp.id);
    setVal('exp-amount', exp.amount);
    setVal('exp-category', exp.category_id);
    setVal('exp-desc', exp.description);
    setVal('exp-date', exp.date.split('T')[0]);
    if (exp.receipt_url) {
        setVal('exp-receipt-url', exp.receipt_url);
        const statusEl = document.getElementById('receipt-status');
        const clearBtn = document.getElementById('clear-receipt');
        if (statusEl) statusEl.innerText = 'Receipt attached';
        if (clearBtn) clearBtn.classList.remove('hidden');
    } else {
        window.clearReceipt();
    }
};
window.deleteExpense = async (id) => {
    const exp = state.expenses.find(e => e.id === id);
    if (!exp) return;

    // Premium Logic: Store in undo buffer instead of immediate server delete
    if (state.undoBuffer.timeout) clearTimeout(state.undoBuffer.timeout);

    state.undoBuffer.action = 'delete';
    state.undoBuffer.data = exp;

    // Optimistic UI: Hide from state and re-render
    state.expenses = state.expenses.filter(e => e.id !== id);
    renderExpensesList(state.expenses);
    updateDashboardMetrics(state.expenses);

    showToast(t('transaction_deleted') || 'Transaction Deleted', 'undo', {
        label: t('undo') || 'Undo',
        callback: () => {
            state.expenses.push(exp);
            state.expenses.sort((a, b) => new Date(b.date) - new Date(a.date));
            renderExpensesList(state.expenses);
            updateDashboardMetrics(state.expenses);
            state.undoBuffer.action = null;
            state.undoBuffer.data = null;
            if (state.undoBuffer.timeout) clearTimeout(state.undoBuffer.timeout);
        }
    });

    state.undoBuffer.timeout = setTimeout(async () => {
        if (state.undoBuffer.action === 'delete') {
            try {
                await apiFetch('/api/expenses', {
                    method: 'DELETE',
                    body: JSON.stringify({ id })
                });
            } catch (err) {
                console.error('Final delete failed:', err);
            }
        }
        state.undoBuffer.action = null;
        state.undoBuffer.data = null;
    }, 5000); // 5 second window for undo
};

/** PREMIUM TOAST NOTIFICATION SYSTEM **/
function showToast(message, type = 'info', action = null) {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;

    let icon = ''; // Professional text-only toast
    if (type === 'success') icon = 'Success: ';
    if (type === 'error') icon = 'Error: ';
    if (type === 'warning') icon = 'Warning: ';
    if (type === 'undo') icon = 'Undo: ';

    toast.innerHTML = `
        <div class="toast-content">
            <div class="toast-message"><strong>${icon}</strong>${message}</div>
        </div>
        <button class="toast-close">Close</button>
    `;

    if (action) {
        const actionBtn = document.createElement('button');
        actionBtn.className = 'toast-btn toast-btn-primary';
        actionBtn.innerText = action.label;
        actionBtn.onclick = () => {
            action.callback();
            toast.remove();
        };
        toast.querySelector('.toast-content').appendChild(actionBtn);
    }

    container.appendChild(toast);

    const timeout = setTimeout(() => {
        toast.classList.add('toast-exit');
        toast.addEventListener('animationend', () => toast.remove());
    }, 5000);

    toast.querySelector('.toast-close').onclick = () => {
        clearTimeout(timeout);
        toast.remove();
    };
}
window.showToast = showToast;

/** BULK SELECTION ENGINE **/
function handleSelectAll(e) {
    const checkboxes = document.querySelectorAll('.expense-checkbox');
    state.bulkSelection.selectedIds.clear();

    checkboxes.forEach(cb => {
        cb.checked = e.target.checked;
        const id = parseInt(cb.dataset.id);
        if (e.target.checked) {
            state.bulkSelection.selectedIds.add(id);
            cb.closest('.list-item').classList.add('expense-row-selected');
        } else {
            cb.closest('.list-item').classList.remove('expense-row-selected');
        }
    });

    updateBulkBar();
}

function updateBulkBar() {
    const checkboxes = document.querySelectorAll('.expense-checkbox:checked');
    state.bulkSelection.selectedIds.clear();
    checkboxes.forEach(cb => state.bulkSelection.selectedIds.add(parseInt(cb.dataset.id)));

    const count = state.bulkSelection.selectedIds.size;
    const bar = document.getElementById('bulk-action-bar');
    const countEl = document.getElementById('selected-count');

    if (count > 0) {
        bar.classList.remove('hidden');
        if (countEl) countEl.innerText = `${count} ${t('selected') || 'selected'}`;
    } else {
        bar.classList.add('hidden');
        if (elements.selectAllExpenses) elements.selectAllExpenses.checked = false;
    }
}

window.clearBulkSelection = () => {
    state.bulkSelection.selectedIds.clear();
    if (elements.selectAllExpenses) elements.selectAllExpenses.checked = false;
    document.querySelectorAll('.expense-checkbox').forEach(cb => {
        cb.checked = false;
        cb.closest('.list-item').classList.remove('expense-row-selected');
    });
    updateBulkBar();
};

window.handleBulkDelete = async () => {
    const ids = Array.from(state.bulkSelection.selectedIds);
    if (ids.length === 0) return;

    if (!confirm(`${t('confirm_bulk_delete') || 'Delete'} ${ids.length} ${t('transactions') || 'transactions'}?`)) return;

    try {
        await apiFetch('/api/expenses/bulk', {
            method: 'DELETE',
            body: JSON.stringify({ ids })
        });

        showToast(`${ids.length} ${t('transactions_deleted') || 'Transactions Deleted'}`, 'success');
        window.clearBulkSelection();
        refreshData();
    } catch (err) {
        showToast(err.message, 'error');
    }
};

window.handleBulkExport = () => {
    const ids = Array.from(state.bulkSelection.selectedIds);
    if (ids.length === 0) return;

    // Redirect to bulk export endpoint with IDs
    const params = new URLSearchParams();
    ids.forEach(id => params.append('ids', id));
    window.location.href = `/api/expenses/export?${params.toString()}`;
};

window.handleSelectAll = handleSelectAll;
window.updateBulkBar = updateBulkBar;

window.restoreExpense = async (id) => {
    if (!confirm(t('confirm_restore_transaction'))) return;
    try {
        await apiFetch('/api/expenses/restore', {
            method: 'POST',
            body: JSON.stringify({ id })
        });
        document.getElementById('trash-item-' + id)?.remove();
        refreshData();
    } catch (err) {
        alert(t('restore_failed') + err.message);
    }
};



async function handleExport() {
    window.location.href = '/api/expenses/export'; // Direct download
}

async function handleViewTrash() {
    try {
        const expenses = await apiFetch('/api/expenses/trash');
        const list = elements.trashList;
        if (!list) return; // Should not happen if state.js is correct

        elements.trashModal.classList.remove('hidden');

        if (expenses.length === 0) {
            list.innerHTML = `<p class="tc text-muted">${t('trash_empty')}</p>`;
            return;
        }

        const symbol = getCurrencySymbol(state.user.currency || 'USD');
        list.innerHTML = expenses.map(e => `
            <div class="list-item" id="trash-item-${e.id}">
                <div class="item-info">
                    <div class="item-title">${escapeHTML(e.description)}</div>
                    <div class="item-meta"><time>${formatDate(e.date)}</time> • ${t('deleted_status')}</div>
                </div>
                 <div class="flex-center" style="gap: 10px;">
                    <div class="item-amount">-${symbol}${e.amount.toFixed(2)}</div>
                    ${e.receipt_url ? `<button class="btn-icon" onclick="window.downloadReceipt('${e.receipt_url}', 'receipt_${e.id}')" title="Download Receipt">Download</button>` : ''}
                    <button class="btn-icon" onclick="window.restoreExpense(${e.id})" title="Restore">Restore</button>
                </div>
            </div>
        `).join('');

    } catch (err) {
        alert('Failed to load trash: ' + err.message);
    }
}

async function handleAdminRegister(e) {
    e.preventDefault();
    const username = val('reg-username');
    const email = val('reg-email');
    const password = val('reg-password');

    try {
        await apiFetch('/api/admin/register', {
            method: 'POST',
            body: JSON.stringify({ username, email, password })
        });
        showToast(t('registration_success'), 'success');
        document.getElementById('admin-register-form').reset();
        refreshAdminData();
    } catch (err) {
        showToast(t('registration_failed') + err.message, 'error');
    }
}

// Ensure refreshAdminData is implemented if expected
async function refreshAdminData() {
    try {
        const users = await apiFetch('/api/admin/users', { skipGlobalErrorHandler: true });
        const container = document.getElementById('users-list-container');
        if (container) {
            container.innerHTML = users.map(u => `
                <div class="flex-center" style="justify-content: space-between; padding: 0.5rem; border-bottom: 1px solid var(--border-color); ${!u.active ? 'background: rgba(255,0,0,0.05);' : ''}">
                    <div>
                        <strong style="${!u.active ? 'text-decoration: line-through; color: #888;' : ''}">${escapeHTML(u.username)}</strong> 
                        <span class="text-muted small">(${escapeHTML(u.role)})</span>
                        ${!u.active ? `<span class="status-badge" style="background: var(--danger-color); font-size: 0.7rem; padding: 2px 4px; border-radius: 4px; color: white; margin-left: 5px;">${t('deleted_status')}</span>` : ''}
                        ${u.id === state.user.id ? `<span class="status-badge" style="background: var(--primary-color); font-size: 0.7rem; padding: 2px 4px; border-radius: 4px; color: white; margin-left: 5px;">${t('you_badge')}</span>` : ''}
                    </div>
                    <div class="flex-center" style="gap: 5px;">
                        ${u.id !== state.user.id ? `
                            <button class="btn-compact" style="background: #3498db; color: white;" onclick="window.impersonateUser(${u.id})">${t('impersonate')}</button>
                            <button class="btn-compact" style="background: #e67e22; color: white;" onclick="window.revokeSessions(${u.id})">${t('revoke_sessions')}</button>
                            ${u.active ?
                        `<button class="btn-delete btn-compact" onclick="window.deleteUser(${u.id})">${t('delete')}</button>` :
                        `<button class="btn-primary btn-compact" style="background: #27ae60;" onclick="window.retrieveUser(${u.id})">${t('retrieve')}</button>`
                    }
                        ` : `<span class="text-muted small">${t('current_admin')}</span>`}
                    </div>
                </div>
            `).join('');
        }

        // Admin Sys Stats & Global Items
        if (state.user.role === 'admin') {
            const [stats, configs, logs] = await Promise.all([
                apiFetch('/api/admin/stats', { skipGlobalErrorHandler: true }),
                apiFetch('/api/admin/configs', { skipGlobalErrorHandler: true }),
                apiFetch('/api/admin/audit-logs', { skipGlobalErrorHandler: true })
            ]);

            if (!stats || !configs || !logs) return;

            const symbol = getCurrencySymbol(state.user.currency || 'USD');

            // Set Configs in UI
            const mMode = configs.find(c => c.key === 'maintenance_mode');
            if (mMode) document.getElementById('maintenance-toggle').checked = mMode.value === 'true';

            const ann = configs.find(c => c.key === 'global_announcement');
            if (ann) setVal('announcement-input', ann.value);

            // Platform Overview (Cards at the top)
            if (elements.adminTotalUsers) elements.adminTotalUsers.innerText = stats.total_users;
            if (elements.adminTotalVolume) elements.adminTotalVolume.innerText = `${symbol}${stats.total_expenses.toFixed(2)}`;

            // Recent Logs & Expenditure Table (inside analytics)


            // User Spending Table
            const spendingTable = document.getElementById('user-spending-table');
            if (spendingTable && stats.user_breakdown) {
                spendingTable.innerHTML = stats.user_breakdown.map(u => `
                    <div class="flex-center" style="justify-content: space-between; border-bottom: 1px solid var(--border-color); padding: 8px;">
                        <span>${escapeHTML(u.username)}</span>
                        <span class="font-weight-900">${symbol}${u.total.toFixed(2)}</span>
                    </div>
                 `).join('');
            }

            // Global Transaction Log
            if (elements.globalExpensesList && !document.getElementById('admin-transaction-log').classList.contains('hidden')) {
                const globalExpenses = await apiFetch('/api/expenses?all=true', { skipGlobalErrorHandler: true });
                if (globalExpenses) {
                    elements.globalExpensesList.innerHTML = globalExpenses.map(e => `
                        <tr>
                            <td class="font-weight-700">${escapeHTML(e.username || 'User ' + e.user_id)}</td>
                            <td>${formatDate(e.date)}</td>
                            <td>${escapeHTML(e.description)}</td>
                            <td class="font-weight-900">${symbol}${e.amount.toFixed(2)}</td>
                             <td>
                                <div class="flex-center" style="gap: 5px;">
                                    ${e.receipt_url ? `<button class="btn-primary btn-compact" onclick="window.downloadReceipt('${e.receipt_url}', 'receipt_${e.id}')" title="Download">Download</button>` : ''}
                                    <button class="btn-delete btn-compact" onclick="window.hardDeleteExpense(${e.id})" title="Delete Forever">Delete</button>
                                </div>
                            </td>
                        </tr>
                    `).join('') || `<tr><td colspan="5" class="tc p-2 text-muted">${t('no_transactions')}</td></tr>`;
                }
            }

            // Logs
            const logsContainer = document.getElementById('admin-audit-logs');
            if (logsContainer && logs) {
                logsContainer.innerHTML = logs.map(l => `
                    <div style="font-size: 0.8rem; margin-bottom: 6px; border-left: 3px solid var(--primary); padding-left: 8px; background: rgba(0,0,0,0.02); border-radius: 0 4px 4px 0; padding-top: 4px; padding-bottom: 4px;">
                        <div class="text-muted small">${formatDate(l.timestamp)}</div> 
                        <div><strong>${escapeHTML(l.action)}</strong>: ${escapeHTML(l.details)}</div>
                    </div>
                 `).join('');
            }
        }
    } catch (e) {
        console.error('Admin data refresh', e);
        showToast(t('admin_data_error') || 'Failed to load admin data: ' + e.message, 'error');
    }
}

window.toggleAdminSection = (sectionId) => {
    const el = document.getElementById(`admin-${sectionId}`);
    if (el) {
        el.classList.toggle('hidden');
        const header = el.previousElementSibling;
        if (header) {
            const arrow = header.querySelector('.dropdown-arrow');
            if (arrow) arrow.style.transform = el.classList.contains('hidden') ? 'rotate(0)' : 'rotate(180deg)';
        }
        if (!el.classList.contains('hidden')) refreshAdminData();
    }
}

// Global toggle for admin user management 
// Global admin user management actions
window.deleteUser = async (id) => {
    if (!confirm(t('delete_user_confirm'))) return;
    try {
        await apiFetch('/api/admin/users', { method: 'DELETE', body: JSON.stringify({ id }) });
        refreshAdminData();
    } catch (e) { alert(e.message); }
};

window.retrieveUser = async (id) => {
    try {
        await apiFetch('/api/admin/users/recover', { method: 'POST', body: JSON.stringify({ id }) });
        refreshAdminData();
    } catch (e) { alert(e.message); }
};

window.hardDeleteUser = async (id) => {
    if (!confirm(t('erase_user_confirm'))) return;
    try {
        await apiFetch('/api/admin/users/hard-delete', { method: 'DELETE', body: JSON.stringify({ id }) });
        refreshAdminData();
    } catch (e) { alert(e.message); }
};

window.hardDeleteExpense = async (id) => {
    if (!confirm('Are you sure you want to PERMANENTLY delete this transaction? This cannot be undone.')) return;
    try {
        await apiFetch('/api/admin/expenses/hard', {
            method: 'DELETE',
            body: JSON.stringify({ id })
        });
        showToast('Transaction deleted forever.', 'success');
        refreshAdminData();
        refreshData();
    } catch (err) {
        showToast('Failed to delete transaction: ' + err.message, 'error');
    }
};

async function handleCurrencyConversion() {
    const amount = parseFloat(val('convert-amount'));
    const from = val('convert-from');
    const to = val('convert-to');
    const resultEl = document.getElementById('conversion-result');

    if (isNaN(amount)) return;

    resultEl.innerText = t('converting');

    try {
        const rates = await getExchangeRates(from);
        if (rates && rates[to]) {
            const result = amount * rates[to];
            const symbol = getCurrencySymbol(to);
            resultEl.innerText = `${amount} ${from} = ${symbol}${result.toFixed(2)} ${to}`;
        } else {
            resultEl.innerText = t('conversion_failed');
        }
    } catch (err) {
        resultEl.innerText = t('conversion_error');
    }
}

// Advanced Admin Functions
window.updateConfig = async (key, value) => {
    try {
        await apiFetch('/api/admin/configs', {
            method: 'POST',
            body: JSON.stringify({ key, value })
        });
        alert(`${t('system_settings')} updated!`);
        if (key === 'global_announcement') {
            const banner = document.getElementById('announcement-banner');
            const text = document.getElementById('announcement-text');
            if (value) {
                text.textContent = value;
                banner.classList.remove('hidden');
            } else {
                banner.classList.add('hidden');
            }
        }
    } catch (err) {
        alert('Failed to update config: ' + err.message);
    }
};

window.revokeSessions = async (userId) => {
    if (!confirm('Are you sure? This will force the user to log in again.')) return;
    try {
        await apiFetch('/api/admin/users/revoke-sessions', {
            method: 'POST',
            body: JSON.stringify({ user_id: userId })
        });
        alert('Sessions revoked successfully!');
    } catch (err) {
        alert('Failed to revoke sessions: ' + err.message);
    }
};

window.impersonateUser = async (userId) => {
    try {
        const { token } = await apiFetch('/api/admin/users/impersonate', {
            method: 'POST',
            body: JSON.stringify({ user_id: userId })
        });

        // Save current token before swapping
        localStorage.setItem('admin_token', localStorage.getItem('token'));
        localStorage.setItem('admin_user', JSON.stringify(state.user));

        // Swap to target user
        localStorage.setItem('token', token);
        // We'll need to fetch user profile to update state properly
        safeReload(); // Simplest way to enter impersonation mode
    } catch (err) {
        alert('Impersonation failed: ' + err.message);
    }
};

window.stopImpersonating = () => {
    const adminToken = localStorage.getItem('admin_token');
    if (!adminToken) return;

    localStorage.setItem('token', adminToken);
    localStorage.setItem('user', localStorage.getItem('admin_user'));
    localStorage.removeItem('admin_token');
    localStorage.removeItem('admin_user');
    safeReload();
};

