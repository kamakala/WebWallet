package handlers

import (
	"log"
	"net/http"
	"text/template"
	"time"
	"webwallet/internal/models"
)

// PageData to struktura do przekazywania danych do szablonów HTML.
type PageData struct {
	Title                string
	Content              string
	PortfolioData        *models.InvestmentPortfolio // Zmieniamy z WalletData na PortfolioData
	MonthlySubsCost      string                      // Sformatowany koszt subskrypcji
	TotalPortfolioValue  string                      // Sformatowana wartość portfela
	ProfitLoss           string                      // Sformatowany zysk/strata
	ProfitLossRaw        float64                     // Sformatowane pod templatke
	ProfitLossPercentage float64                     // Procentowy zysk/strata
}

var tmpl *template.Template

// Globalna instancja portfela inwestycyjnego do celów demonstracyjnych.
var currentPortfolio *models.InvestmentPortfolio

func init() {
	// Definiowanie mapy funkcji dla szablonów
	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"add": func(a, b float64) float64 { // Możesz dodać inne funkcje, np. dodawanie
			return a + b
		},
		// Możesz dodać więcej funkcji pomocniczych tutaj
	}

	var err error
	// Teraz używamy Funcs() do dodania naszych funkcji przed parsowaniem plików
	tmpl, err = template.New("main").Funcs(funcMap).ParseFiles(
		"internal/templates/layout.html",
		"internal/templates/home.html",
	)
	if err != nil {
		log.Fatalf("Error parsing templates: %v", err)
	}

	// Inicjalizacja przykładowego portfela inwestycyjnego
	currentPortfolio = models.NewInvestmentPortfolio()

	// Dodanie przykładowych aktywów
	currentPortfolio.AddAsset(models.Asset{
		ID:       "A1",
		Name:     "Akcje Spółki X",
		Symbol:   "SPX",
		Type:     "Akcje",
		Quantity: 10.0,
		AvgCost:  150.75,
	})
	currentPortfolio.AddAsset(models.Asset{
		ID:       "A2",
		Name:     "Gotówka w PLN",
		Symbol:   "PLN",
		Type:     "Gotówka",
		Quantity: 5000.00,
		AvgCost:  1.0, // Koszt gotówki to 1
	})
	currentPortfolio.AddAsset(models.Asset{
		ID:       "A3",
		Name:     "ETF Globalny",
		Symbol:   "GLO",
		Type:     "ETF",
		Quantity: 5.0,
		AvgCost:  220.00,
	})
	currentPortfolio.AddAsset(models.Asset{
		ID:       "A4",
		Name:     "Obligacje Skarbowe X",
		Symbol:   "OSX",
		Type:     "Obligacje",
		Quantity: 2.0, // Dwie obligacje po 1000 PLN = 2000 PLN
		AvgCost:  1000.00,
	})

	// Dodanie przykładowych subskrypcji
	currentPortfolio.AddSubscription(models.Subscription{
		ID:        "S1",
		Name:      "Platforma Inwestycyjna Pro",
		Cost:      89.99,
		Frequency: "Miesięcznie",
		NextDue:   time.Now().AddDate(0, 1, 0), // Za miesiąc
	})
	currentPortfolio.AddSubscription(models.Subscription{
		ID:        "S2",
		Name:      "Usługa Analityczna Premium",
		Cost:      240.00,
		Frequency: "Rocznie",
		NextDue:   time.Now().AddDate(1, 0, 0), // Za rok
	})

	log.Printf("Portfel inwestycyjny zainicjowany. Całkowita wartość: %.2f PLN\n", currentPortfolio.GetTotalValue())
}

// HomeHandler renderuje stronę główną portfela inwestycyjnego.
func HomeHandler(w http.ResponseWriter, r *http.Request) {

	rawProfitLoss := currentPortfolio.GetProfitLoss()
	// Tymczasowy log do debugowania: sprawdź typ i wartość ProfitLossRaw
	//log.Printf("DEBUG: ProfitLossRaw value: %v, type: %T\n", rawProfitLoss, rawProfitLoss)

	data := PageData{
		Title:                "Mój Portfel Inwestycyjny",
		Content:              "Przegląd Twoich aktywów i kosztów:",
		PortfolioData:        currentPortfolio,
		MonthlySubsCost:      models.FormatCurrency(currentPortfolio.GetMonthlySubscriptionCost()),
		TotalPortfolioValue:  models.FormatCurrency(currentPortfolio.GetTotalValue()),
		ProfitLoss:           models.FormatCurrency(rawProfitLoss),
		ProfitLossRaw:        rawProfitLoss, // Tutaj przypisujemy float64
		ProfitLossPercentage: currentPortfolio.GetProfitLossPercentage(),
	}

	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error executing template: %v", err)
		return // DODANY RETURN
	}
}
