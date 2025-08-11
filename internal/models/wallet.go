package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Asset reprezentuje pojedynczy składnik majątku w portfelu inwestycyjnym.
type Asset struct {
	ID           string  `json:"id" bson:"_id"` // Dodaj tag bson:"_id"
	Name         string  `json:"name" bson:"name"`
	Symbol       string  `json:"symbol" bson:"symbol"`
	Type         string  `json:"type" bson:"type"`
	Quantity     float64 `json:"quantity" bson:"quantity"`
	AvgCost      float64 `json:"avgCost" bson:"avgCost"`
	CurrentPrice float64 `json:"currentPrice" bson:"currentPrice"`
	WalletType   string  `json:"walletType" bson:"walletType"`
}

// Subscription reprezentuje pojedynczą subskrypcję lub stały koszt.
type Subscription struct {
	ID        string    `json:"id" bson:"_id"` // Dodaj tag bson:"_id"
	Name      string    `json:"name" bson:"name"`
	Cost      float64   `json:"cost" bson:"cost"`
	Frequency string    `json:"frequency" bson:"frequency"` // np. "Miesięcznie", "Rocznie"
	NextDue   time.Time `json:"nextDue" bson:"nextDue"`     // Następna data płatności
}

// InvestmentPortfolio reprezentuje cały portfel inwestycyjny użytkownika.
type InvestmentPortfolio struct {
	Assets                  []Asset        // Lista posiadanych aktywów
	Subscriptions           []Subscription // Lista subskrypcji
	TotalValue              float64        // Całkowita szacowana wartość portfela
	TotalCost               float64        // Całkowity koszt zakupu aktywów (bez subskrypcji)
	MonthlySubscriptionCost float64        // Łączny miesięczny koszt subskrypcji
}

// NewInvestmentPortfolio tworzy i zwraca nową instancję pustego portfela inwestycyjnego.
func NewInvestmentPortfolio() *InvestmentPortfolio {
	return &InvestmentPortfolio{
		Assets:                  []Asset{},
		Subscriptions:           []Subscription{},
		TotalValue:              0.0,
		TotalCost:               0.0,
		MonthlySubscriptionCost: 0.0,
	}
}

// AddAsset dodaje nowe aktywo do portfela.
// To uproszczona wersja, która nie uwzględnia aktualizacji istniejących aktywów ani cen rynkowych.
func (p *InvestmentPortfolio) AddAsset(a Asset) {
	p.Assets = append(p.Assets, a)
	// Na razie TotalValue będzie po prostu sumą (Quantity * AvgCost).
	// W przyszłości będziemy pobierać aktualne ceny rynkowe.
	if a.CurrentPrice == 0 {
		a.CurrentPrice = a.AvgCost
	}
	p.TotalValue += a.Quantity * a.CurrentPrice // Uproszczone obliczenie wartości na podstawie średniego kosztu
	p.TotalCost += a.Quantity * a.AvgCost
	p.CalculateTotals() // Przelicz wszystko po dodaniu
}

// AddSubscription dodaje nową subskrypcję do portfela.
func (p *InvestmentPortfolio) AddSubscription(s Subscription) {
	p.Subscriptions = append(p.Subscriptions, s)
	p.CalculateTotals() // Przelicz wszystko po dodaniu
}

// CalculateTotals przelicza sumaryczne wartości portfela.
// POWINNO BYĆ WYWOŁYWANE PO KAŻDEJ ZMIANIE W ASSETACH LUB SUBSKRYPCJACH
func (p *InvestmentPortfolio) CalculateTotals() {
	p.TotalValue = 0.0
	p.TotalCost = 0.0
	p.MonthlySubscriptionCost = 0.0

	for _, a := range p.Assets {
		// Dla uproszczenia, wartość to ilość * średni koszt. W przyszłości będzie to ilość * cena rynkowa.
		p.TotalValue += a.Quantity * a.CurrentPrice
		p.TotalCost += a.Quantity * a.AvgCost
	}

	for _, s := range p.Subscriptions {
		switch s.Frequency {
		case "Miesięcznie":
			p.MonthlySubscriptionCost += s.Cost
		case "Rocznie":
			p.MonthlySubscriptionCost += s.Cost / 12.0
		}
		// Możesz dodać obsługę innych częstotliwości, np. "Kwartalnie"
	}
}

// GetAssets zwraca wszystkie aktywa.
func (p *InvestmentPortfolio) GetAssets() []Asset {
	return p.Assets
}

// GetSubscriptions zwraca wszystkie subskrypcje.
func (p *InvestmentPortfolio) GetSubscriptions() []Subscription {
	return p.Subscriptions
}

// GetTotalValue zwraca całkowitą wartość portfela.
func (p *InvestmentPortfolio) GetTotalValue() float64 {
	p.CalculateTotals() // Upewniamy się, że wartości są aktualne przed zwróceniem
	return p.TotalValue
}

// GetTotalCost zwraca całkowity koszt zakupu aktywów.
func (p *InvestmentPortfolio) GetTotalCost() float64 {
	p.CalculateTotals()
	return p.TotalCost
}

// GetMonthlySubscriptionCost zwraca łączny miesięczny koszt subskrypcji.
func (p *InvestmentPortfolio) GetMonthlySubscriptionCost() float64 {
	p.CalculateTotals()
	return p.MonthlySubscriptionCost
}

// GetProfitLoss oblicza zysk/stratę dla portfela (uproszczone: wartość bieżąca - koszt zakupu).
func (p *InvestmentPortfolio) GetProfitLoss() float64 {
	return p.GetTotalValue() - p.GetTotalCost()
}

// GetProfitLossPercentage oblicza procentowy zysk/stratę.
func (p *InvestmentPortfolio) GetProfitLossPercentage() float64 {
	if p.GetTotalCost() == 0 {
		return 0.0 // Zapobiega dzieleniu przez zero
	}
	return (p.GetProfitLoss() / p.GetTotalCost()) * 100.0
}

// FormatCurrency to pomocnicza funkcja do formatowania kwot.
func FormatCurrency(amount float64) string {
	return fmt.Sprintf("%.2f PLN", amount)
}

func GenerateID() string {
	return uuid.New().String()
}
