// utils.js - Shared utility functions

import { state } from './state.js';

export function t(key) {
    if (!window.translations) return key;
    const lang = (state.user && state.user.language) || 'en';
    const set = window.translations[lang] || window.translations['en'];
    return set[key] || key;
}

export function escapeHTML(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

export function getCurrencySymbol(code) {
    const symbols = {
        'USD': '$', 'EUR': '€', 'GBP': '£', 'JPY': '¥', 'CNY': '¥',
        'KRW': '₩', 'RUB': '₽', 'BRL': 'R$', 'IDR': 'Rp', 'TRY': '₺',
        'SSP': '£', 'UGX': 'USh', 'BDT': '৳', 'PKR': '₨'
    };
    return symbols[code] || '$';
}

export function formatDate(dateStr) {
    if (!dateStr) return '';
    let year, month, day;
    if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) {
        [year, month, day] = dateStr.split('-');
    } else {
        const dateObj = new Date(dateStr);
        if (isNaN(dateObj.getTime())) return dateStr;
        year = dateObj.getFullYear();
        month = String(dateObj.getMonth() + 1).padStart(2, '0');
        day = String(dateObj.getDate()).padStart(2, '0');
    }
    const format = (state.user && state.user.date_format) ? state.user.date_format : 'YYYY-MM-DD';
    if (format === 'DD/MM/YYYY') return `${day}/${month}/${year}`;
    if (format === 'MM/DD/YYYY') return `${month}/${day}/${year}`;
    return `${year}-${month}-${day}`;
}
