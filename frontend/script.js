let financeData = {
    balance: 0,
    savings: 0,
    incomes: [],
    expenses: [],
    yearly_stats: {},
    weekly_stats: { days: Array(7).fill({ incomes: 0, expenses: 0 }) }
};

const balanceEl = document.getElementById('balance');
const savingsEl = document.getElementById('savings');
const savingsInput = document.getElementById('savings-input');
const incomesList = document.getElementById('incomes');
const expensesList = document.getElementById('expenses');
const transactionType = document.getElementById('transaction-type');
const transactionAmount = document.getElementById('transaction-amount');
const transactionNote = document.getElementById('transaction-note');

document.addEventListener('DOMContentLoaded', () => {
    fetchData();
});

async function fetchData() {
    try {
        const response = await fetch('/api/data');
        financeData = await response.json();
        updateUI();
    } catch (error) {
        console.error('Ошибка загрузки данных:', error);
    }
}

function updateUI() {
    balanceEl.textContent = `${financeData.balance.toFixed(2)} ₽`;
    savingsEl.textContent = `${financeData.savings.toFixed(2)} ₽`;
    
    incomesList.innerHTML = '';
    financeData.incomes.forEach(income => {
        const li = document.createElement('li');
        li.innerHTML = `
            <span>${income.note} - ${income.amount.toFixed(2)} ₽</span>
            <small>${income.date}</small>
        `;
        incomesList.appendChild(li);
    });
    
    expensesList.innerHTML = '';
    financeData.expenses.forEach(expense => {
        const li = document.createElement('li');
        li.innerHTML = `
            <span>${expense.note} - ${expense.amount.toFixed(2)} ₽</span>
            <small>${expense.date}</small>
        `;
        expensesList.appendChild(li);
    });
    
    renderCharts();
}

async function addTransaction() {
    const type = transactionType.value;
    const amount = parseFloat(transactionAmount.value);
    const note = transactionNote.value;
    
    if (!amount || amount <= 0 || !note) {
        alert('Пожалуйста, введите корректную сумму и описание');
        return;
    }
    
    const transaction = {
        amount: amount,
        note: note,
        date: new Date().toISOString().split('T')[0]
    };
    
    try {
        const endpoint = type === 'income' ? '/api/add-income' : '/api/add-expense';
        const response = await fetch(endpoint, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(transaction)
        });
        
        if (response.ok) {
            financeData = await response.json();
            updateUI();
            transactionAmount.value = '';
            transactionNote.value = '';
        }
    } catch (error) {
        console.error('Ошибка добавления операции:', error);
    }
}

async function updateSavings() {
    const amount = parseFloat(savingsInput.value);
    
    if (isNaN(amount)) {
        alert('Пожалуйста, введите корректную сумму');
        return;
    }
    
    try {
        const response = await fetch('/api/update-savings', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ amount: amount })
        });
        
        if (response.ok) {
            financeData = await response.json();
            updateUI();
            savingsInput.value = '';
        }
    } catch (error) {
        console.error('Ошибка обновления накоплений:', error);
    }
}

function renderCharts() {
    if (!document.getElementById('weekly-chart') || !document.getElementById('yearly-chart')) {
        console.error('Контейнеры для графиков не найдены');
        return;
    }

    const commonOptions = {
        credits: { enabled: false },
        legend: {
            align: 'center',
            verticalAlign: 'bottom',
            layout: 'horizontal'
        },
        tooltip: {
            shared: true,
            valueSuffix: ' ₽'
        },
        plotOptions: {
            column: {
                grouping: true,
                pointPadding: 0.2,
                groupPadding: 0.1,
                borderWidth: 0
            }
        }
    };

    // Yearly chart
    const currentYear = new Date().getFullYear();
    const months = ['Янв', 'Фев', 'Мар', 'Апр', 'Май', 'Июн', 'Июл', 'Авг', 'Сен', 'Окт', 'Ноя', 'Дек'];
    const yearlyData = financeData.yearly_stats[currentYear] || {};
    
    Highcharts.chart('yearly-chart', Highcharts.merge(commonOptions, {
        title: { text: `Доходы и расходы за ${currentYear} год` },
        xAxis: { categories: months },
        yAxis: { title: { text: 'Сумма (₽)' }, min: 0 },
        series: [
            { name: 'Доходы', data: months.map((_, i) => yearlyData[i+1]?.incomes || 0), color: '#2ecc71' },
            { name: 'Расходы', data: months.map((_, i) => yearlyData[i+1]?.expenses || 0), color: '#e74c3c' }
        ],
        chart: { type: 'column', height: 400 }
    }));
    
    // Weekly chart
    const weekDays = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
    const weeklyData = (financeData.weekly_stats && financeData.weekly_stats.days) || 
                      Array(7).fill({ incomes: 0, expenses: 0 });
    
    Highcharts.chart('weekly-chart', Highcharts.merge(commonOptions, {
        title: { 
            text: `Доходы и расходы за неделю`,
            subtitle: {
                text: `Начало недели: ${financeData.weekly_stats?.start_date || 'не определено'}`,
                style: { fontSize: '12px' }
            }
        },
        xAxis: { categories: weekDays },
        yAxis: { title: { text: 'Сумма (₽)' }, min: 0 },
        series: [
            { name: 'Доходы', data: weeklyData.map(day => day.incomes || 0), color: '#2ecc71' },
            { name: 'Расходы', data: weeklyData.map(day => day.expenses || 0), color: '#e74c3c' }
        ],
        chart: { type: 'column', height: 400 }
    }));
}