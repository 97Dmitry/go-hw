package main

import (
	"encoding/json"
	"fmt"
	"ledger/internal/core/domain"
	"ledger/internal/features/transactions/service"
	"log"
	"time"
)

func main() {
	if err := service.AddTransaction(domain.Transaction{
		ID:          1,
		Amount:      10,
		Category:    "Мороженное",
		Description: "Шоколадное мороженное",
		Date:        time.Now(),
	}); err != nil {
		err := fmt.Errorf("add transaction 1: %v", err)
		fmt.Println(err)
	}

	if err := service.AddTransaction(domain.Transaction{
		ID:          2,
		Amount:      10,
		Category:    "Шторы",
		Description: "Вильветовые шторы",
		Date:        time.Now(),
	}); err != nil {
		err := fmt.Errorf("add transaction 2: %v", err)
		fmt.Println(err)
	}

	transactions := service.ListTransactions()

	data, err := json.MarshalIndent(transactions, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
}
