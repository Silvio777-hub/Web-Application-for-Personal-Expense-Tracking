// charts.js
// Handles all charting logic using Chart.js.

import { state, CATEGORY_COLORS } from './state.js';
import { getCurrencySymbol, t } from './utils.js';

let categoryChart = null;
let trendChart = null;


export function initCharts(allExpenses) {
    initCategoryChart(allExpenses);
    initSpendingTrendChart(allExpenses);

}

export function initCategoryChart(allExpenses) {
    const ctx = document.getElementById('categoryChart');
    if (!ctx) return;

    // Process data
    const totals = {};
    allExpenses.forEach(e => {
        const cat = state.categories.find(c => c.id === e.category_id);
        const name = cat ? cat.name : 'Unknown';
        totals[name] = (totals[name] || 0) + e.amount;
    });

    const labels = Object.keys(totals);
    const data = Object.values(totals);

    if (categoryChart) {
        categoryChart.data.labels = labels;
        categoryChart.data.datasets[0].data = data;
        categoryChart.update();
        return;
    }

    categoryChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: labels,
            datasets: [{
                data: data,
                backgroundColor: CATEGORY_COLORS
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { position: 'bottom', labels: { color: getComputedStyle(document.documentElement).getPropertyValue('--text-color') } }
            }
        }
    });
}

export function initSpendingTrendChart(allExpenses) {
    const ctx = document.getElementById('spendingTrendChart');
    if (!ctx) return;

    // Last 6 months trend
    const months = [];
    const data = [];
    const now = new Date();

    for (let i = 5; i >= 0; i--) {
        const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
        const monthName = d.toLocaleString('default', { month: 'short' });
        months.push(monthName);

        const total = allExpenses.reduce((sum, e) => {
            const ed = new Date(e.date);
            if (ed.getMonth() === d.getMonth() && ed.getFullYear() === d.getFullYear()) {
                return sum + e.amount;
            }
            return sum;
        }, 0);
        data.push(total);
    }

    if (trendChart) {
        trendChart.data.labels = months;
        trendChart.data.datasets[0].data = data;
        trendChart.update();
        return;
    }

    trendChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: months,
            datasets: [{
                label: t('spending_trend'),
                data: data,
                borderColor: '#4f46e5',
                backgroundColor: 'rgba(79, 70, 229, 0.1)',
                fill: true,
                tension: 0.4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    grid: { color: 'rgba(0,0,0,0.05)' },
                    ticks: { color: getComputedStyle(document.documentElement).getPropertyValue('--text-muted') }
                },
                x: {
                    grid: { display: false },
                    ticks: { color: getComputedStyle(document.documentElement).getPropertyValue('--text-muted') }
                }
            },
            plugins: {
                legend: { labels: { color: getComputedStyle(document.documentElement).getPropertyValue('--text-color') } }
            }
        }
    });
}

export function updateChartTheme() {
    const textColor = getComputedStyle(document.documentElement).getPropertyValue('--text-color').trim();
    const textMuted = getComputedStyle(document.documentElement).getPropertyValue('--text-muted').trim();
    const gridColor = document.documentElement.getAttribute('data-theme') === 'dark' ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.05)';

    if (categoryChart) {
        categoryChart.options.plugins.legend.labels.color = textColor;
        categoryChart.update();
    }

    if (trendChart) {
        trendChart.options.scales.x.ticks.color = textMuted;
        trendChart.options.scales.y.ticks.color = textMuted;
        trendChart.options.scales.y.grid.color = gridColor;
        trendChart.options.plugins.legend.labels.color = textColor;
        trendChart.update();
    }
}


