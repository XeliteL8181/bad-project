package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	dataFile = "finance_data.json"
	mutex    = &sync.Mutex{}
)

type FinanceData struct {
	Balance      float64                   `json:"balance"`
	Savings      float64                   `json:"savings"`
	Incomes      []Transaction             `json:"incomes"`
	Expenses     []Transaction             `json:"expenses"`
	YearlyStats  map[int]map[int]MonthStats `json:"yearly_stats"`
	WeeklyStats  WeekStats                 `json:"weekly_stats"`
	LastResetWeek int                      `json:"last_reset_week"`
	LastResetYear int                      `json:"last_reset_year"`
}

type Transaction struct {
	Amount float64 `json:"amount"`
	Date   string  `json:"date"`
	Note   string  `json:"note"`
}

type MonthStats struct {
	Incomes  float64 `json:"incomes"`
	Expenses float64 `json:"expenses"`
}

type WeekStats struct {
	StartDate string      `json:"start_date"`
	Days      [7]DayStats `json:"days"`
}

type DayStats struct {
	Incomes  float64 `json:"incomes"`
	Expenses float64 `json:"expenses"`
}

func main() {
    initData()

    // API роуты
    http.HandleFunc("/api/data", handleData)
    http.HandleFunc("/api/add-income", handleAddIncome)
    http.HandleFunc("/api/add-expense", handleAddExpense)
    http.HandleFunc("/api/update-savings", handleUpdateSavings)

    // Статика
    fs := http.FileServer(http.Dir("../frontend"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))

    // Главная страница
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "../frontend/index.html")
    })

    fmt.Println("Сервер запущен на :8080")
    http.ListenAndServe(":8080", nil)
}


func initData() {
	mutex.Lock()
	defer mutex.Unlock()

	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		now := time.Now()
		data := FinanceData{
			Balance:      0,
			Savings:      0,
			Incomes:      []Transaction{},
			Expenses:     []Transaction{},
			YearlyStats:  make(map[int]map[int]MonthStats),
			WeeklyStats:  WeekStats{StartDate: getStartOfWeek(now).Format("2006-01-02")},
			LastResetWeek: getWeekNumber(now),
			LastResetYear: now.Year(),
		}
		saveData(data)
	}
}

func handleData(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	data := loadData()
	checkAndResetStats(&data)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleAddIncome(w http.ResponseWriter, r *http.Request) {
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
	data.Balance += t.Amount
	data.Incomes = append(data.Incomes, t)
	updateStats(&data, t.Amount, 0, time.Now())

	saveData(data)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

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
	data.Balance -= t.Amount
	data.Expenses = append(data.Expenses, t)
	updateStats(&data, 0, t.Amount, time.Now())

	saveData(data)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

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

func loadData() FinanceData {
	file, err := os.ReadFile(dataFile)
	if err != nil {
		return FinanceData{}
	}

	var data FinanceData
	json.Unmarshal(file, &data)
	return data
}

func saveData(data FinanceData) {
	file, _ := json.MarshalIndent(data, "", " ")
	os.WriteFile(dataFile, file, 0644)
}

func updateStats(data *FinanceData, income, expense float64, date time.Time) {
	// Weekly stats
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 6
	} else {
		weekday -= 1
	}
	data.WeeklyStats.Days[weekday].Incomes += income
	data.WeeklyStats.Days[weekday].Expenses += expense

	// Yearly stats
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

func checkAndResetStats(data *FinanceData) {
	now := time.Now()
	currentYear := now.Year()
	currentWeek := getWeekNumber(now)

	if currentYear != data.LastResetYear {
		data.YearlyStats = make(map[int]map[int]MonthStats)
		data.LastResetYear = currentYear
	}

	if currentWeek != data.LastResetWeek {
		data.WeeklyStats = WeekStats{
			StartDate: getStartOfWeek(now).Format("2006-01-02"),
		}
		data.LastResetWeek = currentWeek
	}
}

func getStartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		return t.AddDate(0, 0, -6)
	}
	return t.AddDate(0, 0, -(weekday-1))
}

func getWeekNumber(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}