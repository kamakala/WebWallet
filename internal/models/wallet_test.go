package models

import (
	"testing" // Importujemy pakiet testing
	"time"
)

// TestNewInvestmentPortfolio sprawdza, czy nowy portfel jest poprawnie inicjalizowany.
func TestNewInvestmentPortfolio(t *testing.T) {
	portfolio := NewInvestmentPortfolio()

	if portfolio == nil {
		t.Errorf("NewInvestmentPortfolio() returned nil, expected a non-nil portfolio")
	}
	if len(portfolio.Assets) != 0 {
		t.Errorf("NewInvestmentPortfolio() expected 0 assets, got %d", len(portfolio.Assets))
	}
	if len(portfolio.Subscriptions) != 0 {
		t.Errorf("NewInvestmentPortfolio() expected 0 subscriptions, got %d", len(portfolio.Subscriptions))
	}
	if portfolio.TotalValue != 0.0 {
		t.Errorf("NewInvestmentPortfolio() expected TotalValue 0.0, got %.2f", portfolio.TotalValue)
	}
	if portfolio.MonthlySubscriptionCost != 0.0 {
		t.Errorf("NewInvestmentPortfolio() expected MonthlySubscriptionCost 0.0, got %.2f", portfolio.MonthlySubscriptionCost)
	}
}

// TestAddAsset sprawdza dodawanie aktywów i aktualizację wartości portfela.
func TestAddAsset(t *testing.T) {
	portfolio := NewInvestmentPortfolio()

	asset1 := Asset{
		ID:       "A1",
		Name:     "Akcje Testowe",
		Symbol:   "TST",
		Type:     "Akcje",
		Quantity: 10.0,
		AvgCost:  100.00,
	}
	portfolio.AddAsset(asset1)

	if len(portfolio.Assets) != 1 {
		t.Errorf("AddAsset() expected 1 asset, got %d", len(portfolio.Assets))
	}
	if portfolio.Assets[0].Name != "Akcje Testowe" {
		t.Errorf("AddAsset() expected asset name 'Akcje Testowe', got '%s'", portfolio.Assets[0].Name)
	}

	// Sprawdź obliczenia wartości
	expectedTotalValue := 10.0 * 100.00 // Quantity * AvgCost
	if portfolio.GetTotalValue() != expectedTotalValue {
		t.Errorf("AddAsset() expected TotalValue %.2f, got %.2f", expectedTotalValue, portfolio.GetTotalValue())
	}
	expectedTotalCost := 10.0 * 100.00
	if portfolio.GetTotalCost() != expectedTotalCost {
		t.Errorf("AddAsset() expected TotalCost %.2f, got %.2f", expectedTotalCost, portfolio.GetTotalCost())
	}

	// Dodaj kolejne aktywo
	asset2 := Asset{
		ID:       "A2",
		Name:     "Gotówka Test",
		Symbol:   "CASH",
		Type:     "Gotówka",
		Quantity: 500.00,
		AvgCost:  1.0,
	}
	portfolio.AddAsset(asset2)

	if len(portfolio.Assets) != 2 {
		t.Errorf("AddAsset() expected 2 assets after adding second, got %d", len(portfolio.Assets))
	}
	expectedTotalValueAfterSecond := (10.0 * 100.00) + (500.00 * 1.0)
	if portfolio.GetTotalValue() != expectedTotalValueAfterSecond {
		t.Errorf("AddAsset() expected TotalValue %.2f, got %.2f", expectedTotalValueAfterSecond, portfolio.GetTotalValue())
	}
	expectedTotalCostAfterSecond := (10.0 * 100.00) + (500.00 * 1.0)
	if portfolio.GetTotalCost() != expectedTotalCostAfterSecond {
		t.Errorf("AddAsset() expected TotalCost %.2f, got %.2f", expectedTotalCostAfterSecond, portfolio.GetTotalCost())
	}
}

// TestAddSubscription sprawdza dodawanie subskrypcji i aktualizację kosztów.
func TestAddSubscription(t *testing.T) {
	portfolio := NewInvestmentPortfolio()

	sub1 := Subscription{
		ID:        "S1",
		Name:      "Miesięczna Sub",
		Cost:      50.00,
		Frequency: "Miesięcznie",
		NextDue:   time.Now(),
	}
	portfolio.AddSubscription(sub1)

	if len(portfolio.Subscriptions) != 1 {
		t.Errorf("AddSubscription() expected 1 subscription, got %d", len(portfolio.Subscriptions))
	}
	if portfolio.Subscriptions[0].Name != "Miesięczna Sub" {
		t.Errorf("AddSubscription() expected subscription name 'Miesięczna Sub', got '%s'", portfolio.Subscriptions[0].Name)
	}

	expectedMonthlyCost := 50.00
	if portfolio.GetMonthlySubscriptionCost() != expectedMonthlyCost {
		t.Errorf("AddSubscription() expected MonthlySubscriptionCost %.2f, got %.2f", expectedMonthlyCost, portfolio.GetMonthlySubscriptionCost())
	}

	// Dodaj subskrypcję roczną
	sub2 := Subscription{
		ID:        "S2",
		Name:      "Roczna Sub",
		Cost:      240.00, // 240 / 12 = 20 miesięcznie
		Frequency: "Rocznie",
		NextDue:   time.Now(),
	}
	portfolio.AddSubscription(sub2)

	if len(portfolio.Subscriptions) != 2 {
		t.Errorf("AddSubscription() expected 2 subscriptions after adding second, got %d", len(portfolio.Subscriptions))
	}

	expectedMonthlyCostAfterSecond := 50.00 + (240.00 / 12.0) // 50 + 20 = 70
	if portfolio.GetMonthlySubscriptionCost() != expectedMonthlyCostAfterSecond {
		t.Errorf("AddSubscription() expected MonthlySubscriptionCost %.2f, got %.2f", expectedMonthlyCostAfterSecond, portfolio.GetMonthlySubscriptionCost())
	}
}

// TestProfitLoss oblicza zysk/stratę i procent.
func TestProfitLoss(t *testing.T) {
	t.Skip("Pominięto TestProfitLoss: wymaga implementacji logiki cen rynkowych (CurrentPrice).")
	portfolio := NewInvestmentPortfolio()

	// Aktywo z zyskiem
	portfolio.AddAsset(Asset{
		ID:       "P1",
		Name:     "Akcje Zysk",
		Symbol:   "PFT",
		Type:     "Akcje",
		Quantity: 10.0,
		AvgCost:  50.0,
	})
	// Aby symulować zysk, potrzebujemy jakiejś "aktualnej wartości".
	// Na razie nasz model oblicza TotalValue na podstawie AvgCost, więc musimy to zaktualizować.
	// W realnej aplikacji mielibyśmy funkcję GetCurrentPrice().
	// Dla potrzeb testu, możemy manipulować TotalValue bezpośrednio lub dodać tymczasowe pole.
	// Na razie testujemy tylko na podstawie AvgCost, więc ProfitLoss będzie 0.
	// W przyszłości, gdy dodasz ceny rynkowe, ten test będzie musiał być rozbudowany.

	// Zgodnie z obecną logiką (TotalValue = TotalCost), zysk/strata zawsze będzie 0.
	// Gdy dodasz pobieranie cen rynkowych, ten test będzie miał sens.
	if portfolio.GetProfitLoss() != 0.0 {
		t.Errorf("GetProfitLoss() expected 0.0, got %.2f (requires market price logic)", portfolio.GetProfitLoss())
	}
	if portfolio.GetProfitLossPercentage() != 0.0 {
		t.Errorf("GetProfitLossPercentage() expected 0.0, got %.2f (requires market price logic)", portfolio.GetProfitLossPercentage())
	}

	// Uproszczony scenariusz, by wymusić zysk/stratę dla testu (tymczasowo)
	// NIE JEST TO OSTATECZNE ROZWIĄZANIE, ale pokazuje, jak działałby test.
	portfolio.TotalValue = 1100.0 // Symulowana wartość rynkowa > koszt
	portfolio.TotalCost = 1000.0  // Symulowany koszt zakupu
	if profitLoss := portfolio.GetProfitLoss(); profitLoss != 100.0 {
		t.Errorf("GetProfitLoss() expected 100.0, got %.2f", profitLoss)
	}
	if profitLossPct := portfolio.GetProfitLossPercentage(); profitLossPct != 10.0 { // 100/1000 * 100
		t.Errorf("GetProfitLossPercentage() expected 10.0, got %.2f", profitLossPct)
	}

	portfolio.TotalValue = 900.0 // Symulowana wartość rynkowa < koszt
	portfolio.TotalCost = 1000.0
	if profitLoss := portfolio.GetProfitLoss(); profitLoss != -100.0 {
		t.Errorf("GetProfitLoss() expected -100.0, got %.2f", profitLoss)
	}
	if profitLossPct := portfolio.GetProfitLossPercentage(); profitLossPct != -10.0 { // -100/1000 * 100
		t.Errorf("GetProfitLossPercentage() expected -10.0, got %.2f", profitLossPct)
	}
}
