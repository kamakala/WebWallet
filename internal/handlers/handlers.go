package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"webwallet/internal/models"
	"webwallet/internal/repository"
	"webwallet/internal/views" // Importujemy pakiet z komponentami templ
)

// PageData (teraz już nie tak potrzebna, ale zostawiamy na razie strukturę danych przekazywaną do widoku)
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

// AppHandler zawiera zależności (np. repozytorium bazy danych).
// `tmpl` nie jest już *template.Template, bo używamy templ.Component.
type AppHandler struct {
	portfolioRepo *repository.PortfolioRepo
}

// NewAppHandler tworzy nową instancję AppHandler z zależnościami.
func NewAppHandler(repo *repository.PortfolioRepo) *AppHandler {
	return &AppHandler{
		portfolioRepo: repo,
	}
}

// HomeHandler renderuje stronę główną portfela inwestycyjnego.
func (h *AppHandler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	portfolio, err := h.portfolioRepo.LoadPortfolio(ctx)
	if err != nil {
		http.Error(w, "Failed to load portfolio", http.StatusInternalServerError)
		log.Printf("Error loading portfolio from DB: %v", err)
		return
	}

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
		if err := h.portfolioRepo.SavePortfolio(ctx, portfolio); err != nil {
			log.Printf("Could not save initial portfolio to DB: %v", err)
		}
	}

	rawProfitLoss := portfolio.GetProfitLoss()

	// Tutaj tworzymy i renderujemy komponenty templ
	homeComponent := views.Home(
		"Przegląd Twoich aktywów i kosztów:",
		portfolio,
		models.FormatCurrency(portfolio.GetMonthlySubscriptionCost()),
		models.FormatCurrency(portfolio.GetTotalValue()),
		models.FormatCurrency(rawProfitLoss),
		rawProfitLoss,
		portfolio.GetProfitLossPercentage(),
	)

	// Renderujemy komponent Home wewnątrz komponentu Layout
	err = views.Layout(
		"Mój Portfel Inwestycyjny", // Tytuł dla Layout
		homeComponent,              // Komponent content
		portfolio,                  // Przekazywanie całego portfela do Layout (jeśli potrzebne w nagłówku/stopce)
		models.FormatCurrency(portfolio.GetMonthlySubscriptionCost()),
		models.FormatCurrency(portfolio.GetTotalValue()),
		models.FormatCurrency(rawProfitLoss),
		rawProfitLoss,
		portfolio.GetProfitLossPercentage(),
	).Render(r.Context(), w)

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error rendering templ component: %v", err)
		return
	}
}

// AddAssetHandler obsługuje dodawanie nowych aktywów.
func (h *AppHandler) AddAssetHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var message string // Komunikat dla użytkownika

	if r.Method == http.MethodPost {
		// Przetwarzanie danych formularza
		err := r.ParseForm()
		if err != nil {
			message = fmt.Sprintf("Błąd parsowania formularza: %v", err)
			log.Printf("Error parsing form: %v", err)
			// Renderuj formularz ponownie z komunikatem o błędzie
			h.renderAddAssetForm(w, message)
			return
		}

		// Pobieranie danych z formularza
		name := r.FormValue("name")
		symbol := r.FormValue("symbol")
		assetType := r.FormValue("type")
		quantityStr := r.FormValue("quantity")
		avgCostStr := r.FormValue("avgCost")

		// Walidacja i konwersja danych
		quantity, err := strconv.ParseFloat(quantityStr, 64)
		if err != nil {
			message = "Nieprawidłowa wartość 'Ilość'."
			h.renderAddAssetForm(w, message)
			return
		}
		avgCost, err := strconv.ParseFloat(avgCostStr, 64)
		if err != nil {
			message = "Nieprawidłowa wartość 'Średni Koszt Zakupu'."
			h.renderAddAssetForm(w, message)
			return
		}

		// Utwórz nowe aktywo
		newAsset := models.Asset{
			ID:       models.GenerateID(), // Utwórz nową funkcję GenerateID w models
			Name:     name,
			Symbol:   symbol,
			Type:     assetType,
			Quantity: quantity,
			AvgCost:  avgCost,
			//CurrentPrice: avgCost, // Na razie CurrentPrice = AvgCost
		}

		// Wczytaj aktualny portfel, dodaj aktywo i zapisz
		portfolio, err := h.portfolioRepo.LoadPortfolio(ctx)
		if err != nil {
			message = fmt.Sprintf("Błąd ładowania portfela: %v", err)
			log.Printf("Error loading portfolio for asset addition: %v", err)
			h.renderAddAssetForm(w, message)
			return
		}

		portfolio.AddAsset(newAsset) // Dodaj nowe aktywo do portfela

		if err := h.portfolioRepo.SavePortfolio(ctx, portfolio); err != nil {
			message = fmt.Sprintf("Błąd zapisu portfela: %v", err)
			log.Printf("Error saving portfolio after asset addition: %v", err)
			h.renderAddAssetForm(w, message)
			return
		}

		message = "Aktywo dodane pomyślnie!"
		log.Printf("Asset added: %+v", newAsset)
		// Przekieruj na stronę główną lub wyświetl formularz z sukcesem
		http.Redirect(w, r, "/", http.StatusSeeOther) // Przekieruj na główną stronę
		return
	}

	// GET request: wyświetl formularz
	h.renderAddAssetForm(w, "")
}

// renderAddAssetForm pomaga renderować komponent AddAssetForm
func (h *AppHandler) renderAddAssetForm(w http.ResponseWriter, message string) {
	err := views.AddAssetForm(message).Render(context.Background(), w)
	if err != nil {
		http.Error(w, "Error rendering add asset form", http.StatusInternalServerError)
		log.Printf("Error rendering add asset form: %v", err)
		return
	}
}
