package main

import (
	"encoding/json" // Для кодирования/декодирования JSON
	"fmt"           // Для вывода сообщений в консоль
	"net/http"      // Для работы с HTTP-сервером
	"os"            // Для работы с файлами
	"sync"          // Для использования мьютекса
	"time"          // Для работы с датой и временем
)

// Глобальные переменные для имени файла и мьютекса (блокировка при доступе к данным)
var (
	dataFile = "finance_data.json" // Имя JSON-файла, где хранятся данные
	mutex    = &sync.Mutex{}       // Мьютекс для защиты параллельного доступа
)

// Структура для хранения всей финансовой информации
// Она сериализуется в JSON и сохраняется в файл
type FinanceData struct {
	Balance       float64                    `json:"balance"`         // Общий баланс
	Savings       float64                    `json:"savings"`         // Накопления
	Incomes       []Transaction              `json:"incomes"`         // Список доходов
	Expenses      []Transaction              `json:"expenses"`        // Список расходов
	YearlyStats   map[int]map[int]MonthStats `json:"yearly_stats"`    // Годовая статистика: год -> месяц -> данные
	WeeklyStats   WeekStats                  `json:"weekly_stats"`    // Недельная статистика
	LastResetWeek int                        `json:"last_reset_week"` // Последняя неделя сброса
	LastResetYear int                        `json:"last_reset_year"` // Последний год сброса
}

// Описание отдельной финансовой операции (доход или расход)
type Transaction struct {
	Amount float64 `json:"amount"` // Сумма
	Date   string  `json:"date"`   // Дата операции
	Note   string  `json:"note"`   // Примечание
}

// Статистика по конкретному месяцу
type MonthStats struct {
	Incomes  float64 `json:"incomes"`  // Сумма доходов за месяц
	Expenses float64 `json:"expenses"` // Сумма расходов за месяц
}

// Статистика за неделю
// Содержит дату начала недели и данные по каждому дню
type WeekStats struct {
	StartDate string      `json:"start_date"` // Дата начала недели (понедельник)
	Days      [7]DayStats `json:"days"`       // Массив из 7 дней (Пн–Вс)
}

// Данные по одному дню
// Используются в недельной статистике
type DayStats struct {
	Incomes  float64 `json:"incomes"`
	Expenses float64 `json:"expenses"`
}

func main() {
	initData() // Инициализация данных при первом запуске

	// Регистрация обработчиков API
	http.HandleFunc("/api/data", handleData)                    // Получить все данные
	http.HandleFunc("/api/add-income", handleAddIncome)         // Добавить доход
	http.HandleFunc("/api/add-expense", handleAddExpense)       // Добавить расход
	http.HandleFunc("/api/update-savings", handleUpdateSavings) // Обновить накопления

	// Раздача статических файлов (HTML, CSS, JS)
	fs := http.FileServer(http.Dir("../frontend"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Главная страница (index.html)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/index.html")
	})

	// Запуск сервера
	fmt.Println("Сервер запущен на :8080")
	http.ListenAndServe(":8080", nil)
}

// Создание начального файла данных, если он отсутствует
func initData() {
	mutex.Lock()
	defer mutex.Unlock()

	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		now := time.Now()
		data := FinanceData{
			Balance:       0,
			Savings:       0,
			Incomes:       []Transaction{},
			Expenses:      []Transaction{},
			YearlyStats:   make(map[int]map[int]MonthStats),
			WeeklyStats:   WeekStats{StartDate: getStartOfWeek(now).Format("2006-01-02")},
			LastResetWeek: getWeekNumber(now),
			LastResetYear: now.Year(),
		}
		saveData(data)
	}
}

// Обработчик: возвращает все данные пользователю
func handleData(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	data := loadData()        // Загружаем данные из файла
	checkAndResetStats(&data) // Проверяем, не пора ли сбросить статистику

	saveData(data)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data) // Отправляем данные клиенту
}

// Обработчик добавления дохода
func handleAddIncome(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if r.Method != "POST" {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var t Transaction
	err := json.NewDecoder(r.Body).Decode(&t) // Декодируем JSON из тела запроса
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data := loadData()
	data.Balance += t.Amount                    // Увеличиваем баланс
	data.Incomes = append(data.Incomes, t)      // Добавляем транзакцию
	updateStats(&data, t.Amount, 0, time.Now()) // Обновляем статистику

	saveData(data) // Сохраняем обновлённые данные
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// Обработчик добавления расхода
func handleAddExpense(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if r.Method != "POST" {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var t Transaction
	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data := loadData()
	data.Balance -= t.Amount                    // Уменьшаем баланс
	data.Expenses = append(data.Expenses, t)    // Добавляем транзакцию
	updateStats(&data, 0, t.Amount, time.Now()) // Обновляем статистику

	saveData(data)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// Обработчик обновления накоплений
func handleUpdateSavings(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if r.Method != "POST" {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Amount float64 `json:"amount"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data := loadData()
	data.Savings = req.Amount
	saveData(data)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// Загрузка данных из JSON-файла
func loadData() FinanceData {
	file, err := os.ReadFile(dataFile)
	if err != nil {
		return FinanceData{} // В случае ошибки возвращаем пустые данные
	}

	var data FinanceData
	json.Unmarshal(file, &data) // Парсим JSON в структуру
	return data
}

// Сохранение данных в JSON-файл
func saveData(data FinanceData) {
	file, _ := json.MarshalIndent(data, "", " ") // Преобразуем структуру в JSON с отступами
	os.WriteFile(dataFile, file, 0644)           // Записываем в файл с правами доступа
}

// Обновление статистики на основе новой транзакции
func updateStats(data *FinanceData, income, expense float64, date time.Time) {
	// Определение дня недели (0 = Пн, 6 = Вс)
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 6 // Вс становится последним днём
	} else {
		weekday -= 1 // Сдвиг: Пн = 0
	}
	// Обновляем недельную статистику
	data.WeeklyStats.Days[weekday].Incomes += income
	data.WeeklyStats.Days[weekday].Expenses += expense

	// Обновляем годовую статистику
	year := date.Year()
	month := int(date.Month())

	if _, exists := data.YearlyStats[year]; !exists {
		data.YearlyStats[year] = make(map[int]MonthStats)
	}
	if _, exists := data.YearlyStats[year][month]; !exists {
		data.YearlyStats[year][month] = MonthStats{}
	}

	stats := data.YearlyStats[year][month]
	stats.Incomes += income
	stats.Expenses += expense
	data.YearlyStats[year][month] = stats
}

// Проверка: наступила ли новая неделя или год для сброса статистики
func checkAndResetStats(data *FinanceData) {
	now := time.Now()
	currentYear := now.Year()
	currentWeek := getWeekNumber(now)

	// Если год сменился — очищаем годовую статистику
	if currentYear != data.LastResetYear {
		data.YearlyStats = make(map[int]map[int]MonthStats)
		data.LastResetYear = currentYear
	}

	// Если неделя сменилась — очищаем недельную статистику
	if currentWeek != data.LastResetWeek {
		data.WeeklyStats = WeekStats{
			StartDate: getStartOfWeek(now).Format("2006-01-02"),
		}
		data.LastResetWeek = currentWeek
	}
}

// Получаем дату начала недели (понедельник)
func getStartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		return t.AddDate(0, 0, -6) // Воскресенье → Понедельник
	}
	return t.AddDate(0, 0, -(weekday - 1))
}

// Получаем номер недели по ISO (1–53)
func getWeekNumber(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}