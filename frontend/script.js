// Скрипт управления финансами (frontend)

// Начальное состояние объекта финансовых данных (если fetch не сработает)
let financeData = {
    balance: 0, // Текущий баланс пользователя
    savings: 0, // Сумма накоплений
    incomes: [], // Массив операций доходов
    expenses: [], // Массив операций расходов
    yearly_stats: {}, // Статистика по месяцам за каждый год
    weekly_stats: { days: Array(7).fill({ incomes: 0, expenses: 0 }) } // Недельная статистика по каждому дню
};

// Получаем элементы интерфейса по их ID
const balanceEl = document.getElementById('balance'); // Элемент для отображения баланса
const savingsEl = document.getElementById('savings'); // Элемент для отображения накоплений
const savingsInput = document.getElementById('savings-input'); // Поле ввода для изменения накоплений
const incomesList = document.getElementById('incomes'); // Список отображения доходов
const expensesList = document.getElementById('expenses'); // Список отображения расходов
const transactionType = document.getElementById('transaction-type'); // Выпадающий список: доход или расход
const transactionAmount = document.getElementById('transaction-amount'); // Поле ввода суммы
const transactionNote = document.getElementById('transaction-note'); // Поле ввода описания операции

// Когда DOM загружен, сразу получаем данные с сервера
document.addEventListener('DOMContentLoaded', () => {
    fetchData(); // Получаем данные с backend
});

// Функция загрузки данных с сервера (GET /api/data)
async function fetchData() {
    try {
        const response = await fetch('/api/data'); // Отправляем GET-запрос к серверу
        financeData = await response.json(); // Преобразуем JSON-ответ в объект
        updateUI(); // Обновляем интерфейс на странице
    } catch (error) {
        console.error('Ошибка загрузки данных:', error); // Выводим ошибку в консоль
    }
}

// Обновление интерфейса на основе полученных данных
function updateUI() {
    // Обновляем текущее значение баланса и накоплений
    balanceEl.textContent = `${financeData.balance.toFixed(2)} ₽`;
    savingsEl.textContent = `${financeData.savings.toFixed(2)} ₽`;

    // Очищаем старый список доходов и создаём новый
    incomesList.innerHTML = '';
    financeData.incomes.forEach(income => {
        const li = document.createElement('li');
        li.innerHTML = `
            <span>${income.note} - ${income.amount.toFixed(2)} ₽</span>
            <small>${income.date}</small>
        `;
        incomesList.appendChild(li);
    });

    // Очищаем старый список расходов и создаём новый
    expensesList.innerHTML = '';
    financeData.expenses.forEach(expense => {
        const li = document.createElement('li');
        li.innerHTML = `
            <span>${expense.note} - ${expense.amount.toFixed(2)} ₽</span>
            <small>${expense.date}</small>
        `;
        expensesList.appendChild(li);
    });

    renderCharts(); // Обновляем графики
}

// Добавление новой операции (доход или расход)
async function addTransaction() {
    const type = transactionType.value; // Получаем выбранный тип операции
    const amount = parseFloat(transactionAmount.value); // Парсим сумму
    const note = transactionNote.value; // Получаем описание

    // Проверка: сумма и описание должны быть корректны
    if (!amount || amount <= 0 || !note) {
        alert('Пожалуйста, введите корректную сумму и описание');
        return;
    }

    const transaction = {
        amount: amount,
        note: note,
        date: new Date().toISOString().split('T')[0] // Текущая дата в формате YYYY-MM-DD
    };

    try {
        // Выбираем конечную точку API в зависимости от типа операции
        const endpoint = type === 'income' ? '/api/add-income' : '/api/add-expense';
        const response = await fetch(endpoint, {
            method: 'POST', // Отправляем POST-запрос
            headers: {
                'Content-Type': 'application/json', // Указываем тип содержимого
            },
            body: JSON.stringify(transaction) // Отправляем данные в формате JSON
        });

        if (response.ok) {
            financeData = await response.json(); // Получаем обновлённые данные
            updateUI(); // Обновляем интерфейс
            transactionAmount.value = ''; // Очищаем поле суммы
            transactionNote.value = ''; // Очищаем поле описания
        }
    } catch (error) {
        console.error('Ошибка добавления операции:', error); // Вывод ошибки
    }
}

// Обновление суммы накоплений
async function updateSavings() {
    const amount = parseFloat(savingsInput.value); // Получаем значение из поля ввода

    // Проверка: должно быть число
    if (isNaN(amount)) {
        alert('Пожалуйста, введите корректную сумму');
        return;
    }

    try {
        const response = await fetch('/api/update-savings', {
            method: 'POST', // Отправляем POST-запрос
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ amount: amount }) // Передаём новую сумму накоплений
        });

        if (response.ok) {
            financeData = await response.json(); // Обновляем данные
            updateUI();
            savingsInput.value = ''; // Очищаем поле ввода
        }
    } catch (error) {
        console.error('Ошибка обновления накоплений:', error); // Логируем ошибку
    }
}

// Построение графиков доходов и расходов
function renderCharts() {
    // Проверяем, что контейнеры для графиков существуют
    if (!document.getElementById('weekly-chart') || !document.getElementById('yearly-chart')) {
        console.error('Контейнеры для графиков не найдены');
        return;
    }

    // Общие настройки для обоих графиков
    const commonOptions = {
        credits: { enabled: false }, // Отключаем логотип Highcharts
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

    // Годовой график
    const currentYear = new Date().getFullYear();
    const months = ['Янв', 'Фев', 'Мар', 'Апр', 'Май', 'Июн', 'Июл', 'Авг', 'Сен', 'Окт', 'Ноя', 'Дек'];
    const yearlyData = financeData.yearly_stats[currentYear] || {}; // Получаем статистику за текущий год

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

    // Недельный график
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