let initialUser = null;
try {
    const saved = localStorage.getItem('user');
    if (saved) initialUser = JSON.parse(saved);
} catch (e) {
    console.error('State: Corrupted user data in LocalStorage. Clearing.', e);
    localStorage.removeItem('user');
}

export const state = {
    user: initialUser,
    isLogin: true,
    isReadOnly: localStorage.getItem('is_read_only') === 'true',
    currentView: 'auth', // 'auth', 'dashboard', 'admin'
    categories: [],
    expenses: [],
    debts: [],
    pagination: {
        page: 1,
        limit: 50,
        hasMore: true
    },
    undoBuffer: {
        action: null,
        data: null,
        timeout: null
    },
    bulkSelection: {
        selectedIds: new Set(),
        isActive: false
    }
};

export const CATEGORY_COLORS = [
    '#06b6d4', // Cyan
    '#f472b6', // Pink
    '#10b981', // Emerald
    '#f59e0b', // Amber
    '#8b5cf6'  // Violet
];

export const elements = {};

export function initElements() {
    const fresh = {
        themeBtn: document.getElementById('theme-btn'),
        authForm: document.getElementById('auth-form'),
        authSection: document.getElementById('auth-section'),
        dashboardSection: document.getElementById('dashboard-section'),
        adminSection: document.getElementById('admin-section'),
        userNav: document.getElementById('user-nav'),
        navCenter: document.querySelector('.nav-center'),
        usernameDisplay: document.getElementById('username-display'),
        logoutBtn: document.getElementById('logout-btn'),
        adminToggleBtn: document.getElementById('admin-toggle-btn'),
        settingsBtn: document.getElementById('settings-btn'),
        settingsModal: document.getElementById('settings-modal'),
        settingsForm: document.getElementById('settings-form'),
        passwordForm: document.getElementById('password-form'),
        preferencesForm: document.getElementById('preferences-form'),
        profilePicInput: document.getElementById('profile-pic-input'),
        profilePreview: document.getElementById('profile-preview'),
        navAvatar: document.getElementById('nav-avatar'),
        navAvatarPlaceholder: document.getElementById('nav-avatar-placeholder'),
        expenseModal: document.getElementById('expense-modal'),
        addExpenseBtn: document.getElementById('add-expense-btn'),
        closeModal: document.getElementById('close-modal'),
        expenseForm: document.getElementById('expense-form'),
        adminRegisterForm: document.getElementById('admin-register-form'),
        userCountBadge: document.getElementById('user-count-badge'),
        forgotPassLink: document.getElementById('forgot-password-link'),
        resetModal: document.getElementById('reset-modal'),
        resetRequestForm: document.getElementById('reset-request-form'),
        resetVerifyForm: document.getElementById('reset-verify-form'),
        resetStep1: document.getElementById('reset-step-1'),
        resetStep2: document.getElementById('reset-step-2'),
        expenseFilter: document.getElementById('expense-filter'),
        exportBtn: document.getElementById('export-btn'),
        customDatePicker: document.getElementById('custom-date-picker'),
        filterStartDate: document.getElementById('filter-start-date'),
        filterEndDate: document.getElementById('filter-end-date'),
        applyFilterBtn: document.getElementById('apply-filter-btn'),
        topCategoriesList: document.getElementById('top-categories-list'),
        savingsBalanceDisplay: document.getElementById('savings-balance'),
        savingsStatusDisplay: document.getElementById('savings-status'),
        setSavingsGoal: document.getElementById('set-monthly-savings-goal'),
        navAddExpenseBtn: document.getElementById('nav-add-expense-btn'),
        expenseSearch: document.getElementById('expense-search'),
        filterMinAmount: document.getElementById('filter-min-amount'),
        filterMaxAmount: document.getElementById('filter-max-amount'),
        savingsProgressBar: document.getElementById('savings-progress-bar'),
        savingsPercent: document.getElementById('savings-percent'),
        monthSpending: document.getElementById('month-spending'),
        monthSpendingProgress: document.getElementById('month-spending-progress'),
        viewTrashBtn: document.getElementById('view-trash-btn'),
        trashModal: document.getElementById('trash-modal'),
        trashList: document.getElementById('trash-list'),
        adminTotalUsers: document.getElementById('admin-total-users'),
        adminTotalVolume: document.getElementById('admin-total-volume'),
        globalExpensesList: document.getElementById('global-expenses-list'),
        impersonationBanner: document.getElementById('impersonation-banner')


    };

    // Mutate the original exported object to keep references live
    Object.assign(elements, fresh);
    console.log('Elements mapping initialized smoothly.');
}

