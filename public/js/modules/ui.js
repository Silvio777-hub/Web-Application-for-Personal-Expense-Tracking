// ui.js
// Handles DOM updates, rendering, and theme logic.

import { state, elements, CATEGORY_COLORS } from './state.js';
import { updateChartTheme } from './charts.js';
import { t, escapeHTML, getCurrencySymbol, formatDate } from './utils.js';

export { t, escapeHTML, getCurrencySymbol, formatDate };

export function setupTheme() {
    const savedTheme = localStorage.getItem('theme');

    // Auto-detection logic for professional feel
    if (!savedTheme) {
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        const theme = prefersDark ? 'dark' : 'light';
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('theme', theme);
    } else {
        document.documentElement.setAttribute('data-theme', savedTheme);
    }

    if (elements.themeBtn) {
        elements.themeBtn.addEventListener('click', () => {
            const currentTheme = document.documentElement.getAttribute('data-theme');
            const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
            document.documentElement.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
            updateChartTheme();
        });
    }

    // Listen for system theme changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', e => {
        if (!localStorage.getItem('theme_manual')) { // Only auto-switch if user hasn't manually set it
            const newTheme = e.matches ? 'dark' : 'light';
            document.documentElement.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
            updateChartTheme();
        }
    });
}

export function translateUI(lang) {
    if (!window.translations) return;
    const translationsSet = window.translations[lang] || window.translations['en'];
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        if (translationsSet[key]) {
            if (el.tagName === 'INPUT' && el.type !== 'submit') {
                el.placeholder = translationsSet[key];
            } else {
                el.innerText = translationsSet[key];
            }
        }
    });

    // Handle role select labels specifically if needed
    const roleSelect = document.getElementById('role-select');
    if (roleSelect && roleSelect.options.length >= 2) {
        roleSelect.options[0].text = translationsSet.regular_user || 'Regular User';
        roleSelect.options[1].text = translationsSet.administrator || 'Administrator';
    }
}



export function getCategoryColor(id) {
    if (!id) return 'var(--border-color)';
    return CATEGORY_COLORS[id % CATEGORY_COLORS.length];
}

export function renderExpensesList(expenses) {
    if (!elements.expensesList) elements.expensesList = document.getElementById('expenses-list');
    const list = elements.expensesList;
    if (!list) return;

    const symbol = getCurrencySymbol(state.user.currency || 'USD');

    if (!expenses || expenses.length === 0) {
        list.innerHTML = `<div class="tc p-4 text-muted" role="status" data-i18n="no_transactions">${t('no_transactions')}</div>`;
        return;
    }

    list.innerHTML = expenses.map(e => {
        const cat = state.categories.find(c => c.id === e.category_id) || { name: 'Uncategorized', color: '#ccc' };
        const symbol = getCurrencySymbol(e.currency);
        const isSelected = state.bulkSelection.selectedIds.has(e.id);

        return `
        <div class="list-item animate-slide-in ${isSelected ? 'expense-row-selected' : ''}" id="expense-${e.id}">
             <div class="flex-center" style="gap: 12px;">
                <input type="checkbox" class="expense-checkbox" data-id="${e.id}" style="width: 18px; height: 18px; cursor: pointer;" ${isSelected ? 'checked' : ''}>
                <div class="category-icon" style="background: ${cat.color}22; color: ${cat.color}">${cat.name[0]}</div>
                <div class="item-info">
                    <div class="item-title">${escapeHTML(e.description) || t('no_description')}</div>
                    <div class="item-meta">
                        <span class="category-tag" style="background: ${cat.color}22; color: ${cat.color}">${cat.name}</span>
                        <time>${formatDate(e.date)}</time>
                    </div>
                </div>
            </div>
            <div class="flex-center" style="gap: 15px;">
                <div class="item-amount" aria-label="Amount">-${symbol}${e.amount.toFixed(2)}</div>
                <div class="item-actions">
                    ${e.receipt_url ? `<button class="btn-download" onclick="window.downloadReceipt('${e.receipt_url}', 'receipt_${e.id}')" title="Download Receipt">Download</button>` : ''}
                    <button class="btn-edit" onclick="window.editExpense(${e.id})" aria-label="Edit transaction">Edit</button>
                    <button class="btn-delete" onclick="window.deleteExpense(${e.id})" aria-label="Delete transaction">Delete</button>
                </div>
            </div>
        </div>
        `;
    }).join('');

    if (state.pagination.hasMore) {
        list.innerHTML += `
            <div class="tc mt-2 mb-2">
                <button id="load-more-btn" class="btn btn-secondary small" onclick="window.loadMoreExpenses()" data-i18n="load_more">${t('load_more')}</button>
            </div>
        `;
    }
}

export function renderTopLowCategories(expenses) {
    if (!elements.topCategoriesList) return;

    const totals = {};
    expenses.forEach(e => {
        const cat = state.categories.find(c => c.id === e.category_id);
        const name = cat ? cat.name : 'Unknown';
        totals[name] = (totals[name] || 0) + e.amount;
    });

    const sorted = Object.entries(totals).sort((a, b) => b[1] - a[1]);
    const symbol = getCurrencySymbol(state.user.currency || 'USD');

    if (sorted.length === 0) {
        elements.topCategoriesList.innerHTML = `<p class="tc p-2 text-muted">${t('no_data_available')}</p>`;
        return;
    }

    let html = `<div class="insight-label mb-1" data-i18n="highest_spends">${t('highest_spends')}</div>`;
    html += sorted.slice(0, 2).map(([name, total]) => `
        <div class="flex-center" style="justify-content: space-between; padding: 0.5rem; border-bottom: 1px solid var(--border-color);">
            <span class="font-weight-500">${escapeHTML(name)}</span>
            <span class="font-weight-900">${symbol}${total.toFixed(2)}</span>
        </div>
    `).join('');

    if (sorted.length > 2) {
        html += `<div class="insight-label mt-1 mb-1" data-i18n="lowest_spends">${t('lowest_spends')}</div>`;
        html += sorted.slice(-2).reverse().map(([name, total]) => `
            <div class="flex-center" style="justify-content: space-between; padding: 0.5rem; border-bottom: 1px solid var(--border-color);">
                <span class="font-weight-500">${escapeHTML(name)}</span>
                <span class="font-weight-900">${symbol}${total.toFixed(2)}</span>
            </div>
        `).join('');
    }

    elements.topCategoriesList.innerHTML = html;
}

export function updateGoalStatus(allExpenses) {
    const symbol = getCurrencySymbol(state.user.currency || 'USD');
    const now = new Date();
    const curMonth = now.getMonth();
    const curYear = now.getFullYear();

    const monthlyTotal = allExpenses.reduce((sum, e) => {
        const d = new Date(e.date);
        return (d.getMonth() === curMonth && d.getFullYear() === curYear) ? sum + e.amount : sum;
    }, 0);

    const goal = state.user.monthly_savings_goal || 0;
    const balance = goal - monthlyTotal;

    // 1. Update Percent Text
    if (elements.savingsPercent) {
        const percent = goal > 0 ? ((monthlyTotal / goal) * 100) : 0;
        elements.savingsPercent.innerText = `${Math.min(100, percent).toFixed(1)}%`;
        if (monthlyTotal > goal) elements.savingsPercent.classList.add('text-danger');
        else elements.savingsPercent.classList.remove('text-danger');
    }

    // 2. Update Progress Bar
    if (elements.savingsProgressBar) {
        let percent = goal > 0 ? ((monthlyTotal / goal) * 100) : 0;
        percent = Math.min(100, percent);
        elements.savingsProgressBar.style.width = `${percent}%`;
        elements.savingsProgressBar.className = 'progress'; // Reset

        if (monthlyTotal > goal) {
            elements.savingsProgressBar.classList.add('danger');
        } else if (percent >= 80) {
            elements.savingsProgressBar.classList.add('warning');
        } else {
            elements.savingsProgressBar.classList.add('success');
        }
    }

    // 3. Update Detailed Math Display
    const goalEl = document.getElementById('math-goal');
    const spentEl = document.getElementById('math-spent');
    const remEl = document.getElementById('math-remaining');

    if (goalEl) goalEl.innerText = `${symbol}${goal.toFixed(2)}`;
    if (spentEl) spentEl.innerText = `${symbol}${monthlyTotal.toFixed(2)}`;
    if (remEl) {
        remEl.innerText = `${symbol}${balance.toFixed(2)}`;
        remEl.className = 'font-weight-900 ' + (balance < 0 ? 'text-danger' : 'text-success');
    }

    // Legacy / Other UI (keep for backward compatibility)
    if (elements.savingsBalanceDisplay) {
        elements.savingsBalanceDisplay.innerText = `${symbol}${balance.toFixed(2)}`;
        elements.savingsBalanceDisplay.className = 'amount' + (balance < 0 ? ' text-danger' : '');
    }

    if (elements.savingsStatusDisplay) {
        if (goal === 0) {
            elements.savingsStatusDisplay.innerText = t('set_savings_goal_msg');
        } else if (balance < 0) {
            elements.savingsStatusDisplay.innerText = t('goal_exceeded');
        } else {
            elements.savingsStatusDisplay.innerText = t('on_track');
        }
    }

    // Update Month Spending Widget
    if (elements.monthSpendingProgress) {
        const percent = (goal > 0 && monthlyTotal > 0) ? Math.min(100, (monthlyTotal / goal) * 100) : 0;
        elements.monthSpendingProgress.style.width = `${percent}%`;
    }

    if (elements.monthSpending) {
        elements.monthSpending.innerText = `${symbol}${monthlyTotal.toFixed(2)}`;
    }
}
