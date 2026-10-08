package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
	"github.com/97Dmitry/go-hw/ledger/internal/repository/memory"
	"github.com/97Dmitry/go-hw/ledger/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type BudgetJSONPeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type BudgetJSON struct {
	Category string           `json:"category"`
	Limit    float64          `json:"limit"`
	Period   BudgetJSONPeriod `json:"period"`
}

type Validatable interface {
	Validate() error
}

func CheckValid(v Validatable) error {
	return v.Validate()
}

func run() error {
	budgetRepo := memory.NewBudgetRepository()
	budgetService := service.NewBudgetService(budgetRepo)

	transactionRepo := memory.NewTransactionRepository()
	transactionService := service.NewTransactionService(transactionRepo, budgetService)

	period := domain.MonthPeriod(time.Now())

	fileBytes, err := os.ReadFile("budgets.json")
	if err != nil {
		return fmt.Errorf("could not read budgets.json: %w", err)
	}

	var budgetSeedJSON []BudgetJSON

	err = json.Unmarshal(fileBytes, &budgetSeedJSON)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	for _, budgetJSON := range budgetSeedJSON {
		var period domain.Period
		if start, err := time.Parse("2006-01-02", budgetJSON.Period.Start); err != nil {
			panic("could not parse budget period start")
		} else {
			period.Start = start
			period.End = start.AddDate(0, 1, 0)
		}

		budget := domain.Budget{
			Category: budgetJSON.Category,
			Limit:    budgetJSON.Limit,
			Period:   period,
		}

		if err := CheckValid(&budget); err != nil {
			return fmt.Errorf("check budget valid %w", err)
		}

		if _, err := budgetService.Add(budget); err != nil {
			return fmt.Errorf("add budget %q: %w", budget.Category, err)
		}
	}

	// Выводим ошибку, что бюджет уже существует в этом месяце
	if _, err := budgetService.Add(domain.Budget{
		Category: "ЖКХ",
		Limit:    1000.00,
		Period:   period,
	}); err != nil {
		fmt.Printf("Планируемая ошибка - add existing budget %q: %v\n", "ЖКХ", err)
	}

	transactionSeed := []domain.Transaction{
		{
			Amount:      1000,
			Category:    "Продукты",
			Description: "Шоколадное мороженное",
			Date:        time.Now(),
		},
		{
			Amount:      1000,
			Category:    "ЖКХ",
			Description: "Электричество",
			Date:        time.Now(),
		},
	}

	for _, tx := range transactionSeed {
		if err := CheckValid(&tx); err != nil {
			return fmt.Errorf("check transaction valid %w", err)
		}

		if _, err := transactionService.AddTransaction(tx); err != nil {
			return fmt.Errorf("add transaction %q: %w", tx.Description, err)
		}
	}

	// Выводим ошибку выхода за бюджет
	if _, err := transactionService.AddTransaction(domain.Transaction{
		Amount:      2000,
		Category:    "Продукты",
		Description: "Вино",
		Date:        time.Now(),
	}); err != nil {
		fmt.Printf("Планируемая ошибка - add transaction %q: %v\n\n", "Вино", err)
	}

	transactions, err := json.MarshalIndent(transactionService.ListTransactions(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal transactions: %w", err)
	}

	budgets, err := json.MarshalIndent(budgetService.List(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal budgets: %w", err)
	}

	fmt.Println("Транзакции:\n", string(transactions))
	fmt.Println("Бюджеты:\n", string(budgets))

	return nil
}
