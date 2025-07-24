package handlers

import (
	"context" // Potrzebne do kontekstu dla operacji DB
	"log"
	"net/http"
	"text/template"
	"time"
	"webwallet/internal/models"
	"webwallet/internal/repository" // Importujemy repozytorium
)

// PageData to struktura do przekazywania danych do szablonów HTML.
type PageData struct {
	Title                string
	Content              string
	PortfolioData        *models.InvestmentPortfolio
	MonthlySubsCost      string
	TotalPortfolioValue  string
	ProfitLoss           string
	ProfitLossRaw        float64
	ProfitLossPercentage float64
}

// AppHandler zawiera zależności (np. repozytorium bazy danych)
type AppHandler struct {
	tmpl          *template.Template
	portfolioRepo *repository.PortfolioRepo // Repozytorium jest teraz polem w AppHandler
}

// NewAppHandler tworzy nową instancję AppHandler z zależnościami.
func NewAppHandler(repo *repository.PortfolioRepo) *AppHandler {
	// Definiowanie mapy funkcji dla szablonów
	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"add": func(a, b float64) float64 {
			return a + b
		},
	}

	// Inicjalizacja szablonów (teraz w NewAppHandler, a nie w init())
	// Daje to większą kontrolę nad szablonami, np. ich przeładowywaniem.
	parsedTmpl, err := template.New("main").Funcs(funcMap).ParseFiles(
		"internal/templates/layout.html",
		"internal/templates/home.html",
	)
	if err != nil {
		log.Fatalf("Error parsing templates: %v", err)
	}

	return &AppHandler{
		tmpl:          parsedTmpl,
		portfolioRepo: repo, // Wstrzykujemy repozytorium
	}
}

// HomeHandler renderuje stronę główną portfela inwestycyjnego.
// Teraz jest to metoda typu AppHandler.
func (h *AppHandler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second) // Kontekst dla operacji DB
	defer cancel()

	// Ładujemy portfel z bazy danych
	portfolio, err := h.portfolioRepo.LoadPortfolio(ctx)
	if err != nil {
		http.Error(w, "Failed to load portfolio", http.StatusInternalServerError)
		log.Printf("Error loading portfolio from DB: %v", err)
		return
	}

	// Jeśli portfel jest nowy/pusty, inicjalizujemy go przykładowymi danymi i zapisujemy.
	// Robimy to tylko raz, przy pierwszym uruchomieniu i braku danych w DB.
	if len(portfolio.Assets) == 0 && len(portfolio.Subscriptions) == 0 {
		log.Println("Database is empty, populating with initial sample data...")
		portfolio.AddAsset(models.Asset{
			ID:       "A1",
			Name:     "Akcje Testowe (DB)",
			Symbol:   "DBG",
			Type:     "Akcje",
			Quantity: 10.0,
			AvgCost:  100.00,
		})
		portfolio.AddSubscription(models.Subscription{
			ID:        "S1",
			Name:      "Miesięczna Sub (DB)",
			Cost:      50.00,
			Frequency: "Miesięcznie",
			NextDue:   time.Now().AddDate(0, 1, 0),
		})
		// Zapisz początkowy portfel do bazy danych
		if err := h.portfolioRepo.SavePortfolio(ctx, portfolio); err != nil {
			log.Printf("Could not save initial portfolio to DB: %v", err)
		}
	}

	rawProfitLoss := portfolio.GetProfitLoss()

	data := PageData{
		Title:                "Mój Portfel Inwestycyjny",
		Content:              "Przegląd Twoich aktywów i kosztów:",
		PortfolioData:        portfolio, // Przekazujemy załadowany portfel
		MonthlySubsCost:      models.FormatCurrency(portfolio.GetMonthlySubscriptionCost()),
		TotalPortfolioValue:  models.FormatCurrency(portfolio.GetTotalValue()),
		ProfitLoss:           models.FormatCurrency(rawProfitLoss),
		ProfitLossRaw:        rawProfitLoss,
		ProfitLossPercentage: portfolio.GetProfitLossPercentage(),
	}

	err = h.tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error executing template: %v", err)
		return
	}
}
