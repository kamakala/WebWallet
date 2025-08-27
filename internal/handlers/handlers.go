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

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
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

// ThemeToggleHandler zmienia wartość motywu w ciasteczku.
func (h *AppHandler) ThemeToggleHandler(w http.ResponseWriter, r *http.Request) {
	currentTheme := "light"
	cookie, err := r.Cookie("theme")
	if err == nil && cookie.Value == "dark" {
		currentTheme = "dark"
	}

	newTheme := "dark"
	if currentTheme == "dark" {
		newTheme = "light"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "theme",
		Value:    newTheme,
		Path:     "/",             // Ciasteczko dostępne na całej stronie
		MaxAge:   3600 * 24 * 365, // Ważne przez rok
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Przekieruj użytkownika z powrotem na stronę, z której przyszedł.
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
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
			ID:         "A1",
			Name:       "Akcje Testowe (DB)",
			Symbol:     "DBG",
			Type:       "Akcje",
			Quantity:   10.0,
			AvgCost:    100.00,
			WalletType: "Poduszka",
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
			h.renderAddAssetForm(w, r, message)
			return
		}

		// Pobieranie danych z formularza
		name := r.FormValue("name")
		symbol := r.FormValue("symbol")
		assetType := r.FormValue("type")
		quantityStr := r.FormValue("quantity")
		avgCostStr := r.FormValue("avgCost")
		currentPriceStr := r.FormValue("currentPrice")
		walletType := r.FormValue("walletType")

		// Walidacja i konwersja danych
		quantity, err := strconv.ParseFloat(quantityStr, 64)
		if err != nil {
			message = "Nieprawidłowa wartość 'Ilość'."
			h.renderAddAssetForm(w, r, message)
			return
		}
		avgCost, err := strconv.ParseFloat(avgCostStr, 64)
		if err != nil {
			message = "Nieprawidłowa wartość 'Średni Koszt Zakupu'."
			h.renderAddAssetForm(w, r, message)
			return
		}

		currentPrice, err := strconv.ParseFloat(currentPriceStr, 64)
		if err != nil {
			message = "Nieprawidłowa wartość 'Obecna cena'."
			h.renderAddAssetForm(w, r, message)
			return
		}

		// Utwórz nowe aktywo
		newAsset := models.Asset{
			ID:           models.GenerateID(), // Utwórz nową funkcję GenerateID w models
			Name:         name,
			Symbol:       symbol,
			Type:         assetType,
			Quantity:     quantity,
			AvgCost:      avgCost,
			CurrentPrice: currentPrice,
			WalletType:   walletType,
		}

		// Wczytaj aktualny portfel, dodaj aktywo i zapisz
		portfolio, err := h.portfolioRepo.LoadPortfolio(ctx)
		if err != nil {
			message = fmt.Sprintf("Błąd ładowania portfela: %v", err)
			log.Printf("Error loading portfolio for asset addition: %v", err)
			h.renderAddAssetForm(w, r, message)
			return
		}

		portfolio.AddAsset(newAsset) // Dodaj nowe aktywo do portfela

		if err := h.portfolioRepo.SavePortfolio(ctx, portfolio); err != nil {
			message = fmt.Sprintf("Błąd zapisu portfela: %v", err)
			log.Printf("Error saving portfolio after asset addition: %v", err)
			h.renderAddAssetForm(w, r, message)
			return
		}

		message = "Aktywo dodane pomyślnie!"
		log.Printf("Asset added: %+v", newAsset)
		// Przekieruj na stronę główną lub wyświetl formularz z sukcesem
		http.Redirect(w, r, "/", http.StatusSeeOther) // Przekieruj na główną stronę
		return
	}

	// GET request: wyświetl formularz
	h.renderAddAssetForm(w, r, "")
}

// renderAddAssetForm pomaga renderować komponent AddAssetForm
func (h *AppHandler) renderAddAssetForm(w http.ResponseWriter, r *http.Request, message string) {
	err := views.AddAssetForm(message).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Error rendering add asset form", http.StatusInternalServerError)
		log.Printf("Error rendering add asset form: %v", err)
		return
	}
}

// DeleteAssetHandler usuwa aktywo z portfela.
func (h *AppHandler) DeleteAssetHandler(w http.ResponseWriter, r *http.Request) {
	// Sprawdź, czy żądanie jest metodą POST (dla bezpieczeństwa i dobrych praktyk)
	if r.Method != http.MethodPost {
		http.Error(w, "Metoda niedozwolona", http.StatusMethodNotAllowed)
		return
	}

	assetID := r.URL.Query().Get("id") // Pobieramy ID aktywa z parametru zapytania URL
	if assetID == "" {
		http.Error(w, "Brak identyfikatora aktywa", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err := h.portfolioRepo.RemoveAsset(ctx, assetID)
	if err != nil {
		log.Printf("Błąd usuwania aktywa (ID: %s): %v", assetID, err)
		http.Error(w, fmt.Sprintf("Nie udało się usunąć aktywa: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Aktywo o ID %s usunięte pomyślnie.", assetID)
	http.Redirect(w, r, "/", http.StatusSeeOther) // Przekieruj z powrotem na stronę główną
}

// UpdateAssetHandler obsługuje aktualizację istniejących aktywów.
func (h *AppHandler) UpdateAssetHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var message string           // Komunikat dla użytkownika
	var targetAsset models.Asset // Aktywo, które będziemy aktualizować/wyświetlać

	if r.Method == http.MethodPost {
		// --- Obsługa żądania POST (przetwarzanie formularza) ---
		err := r.ParseForm()
		if err != nil {
			message = fmt.Sprintf("Błąd parsowania formularza: %v", err)
			log.Printf("Error parsing update asset form: %v", err)
			// Spróbuj załadować aktywo, żeby formularz nie był pusty
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, a := range portfolio.Assets {
					if a.ID == r.FormValue("asset_id") {
						targetAsset = a
						break
					}
				}
			}
			h.renderUpdateAssetForm(w, r, targetAsset, message)
			return
		}

		assetID := r.FormValue("asset_id")
		additionalQuantityStr := r.FormValue("additional_quantity")
		newPurchasePriceStr := r.FormValue("new_purchase_price")

		if assetID == "" {
			message = "Brak identyfikatora aktywa do aktualizacji."
			h.renderUpdateAssetForm(w, r, targetAsset, message) // targetAsset będzie puste
			return
		}

		additionalQuantity, err := strconv.ParseFloat(additionalQuantityStr, 64)
		if err != nil || additionalQuantity <= 0 {
			message = "Nieprawidłowa wartość 'Dodatkowa Ilość'. Musi być liczbą większą od zera."
			// Spróbuj załadować aktywo, żeby formularz nie był pusty
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, a := range portfolio.Assets {
					if a.ID == assetID {
						targetAsset = a
						break
					}
				}
			}
			h.renderUpdateAssetForm(w, r, targetAsset, message)
			return
		}

		newPurchasePrice, err := strconv.ParseFloat(newPurchasePriceStr, 64)
		if err != nil || newPurchasePrice <= 0 {
			message = "Nieprawidłowa wartość 'Cena Zakupu dla Nowej Ilości'. Musi być liczbą większą od zera."
			// Spróbuj załadować aktywo, żeby formularz nie był pusty
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, a := range portfolio.Assets {
					if a.ID == assetID {
						targetAsset = a
						break
					}
				}
			}
			h.renderUpdateAssetForm(w, r, targetAsset, message)
			return
		}

		// Wywołaj funkcję repozytorium do aktualizacji aktywa
		err = h.portfolioRepo.UpdateAsset(ctx, assetID, additionalQuantity, newPurchasePrice)
		if err != nil {
			message = fmt.Sprintf("Błąd aktualizacji aktywa: %v", err)
			log.Printf("Error updating asset (ID: %s): %v", assetID, err)
			// Spróbuj załadować aktywo, żeby formularz nie był pusty
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, a := range portfolio.Assets {
					if a.ID == assetID {
						targetAsset = a
						break
					}
				}
			}
			h.renderUpdateAssetForm(w, r, targetAsset, message)
			return
		}

		log.Printf("Aktywo o ID %s zaktualizowane pomyślnie. Dodano %.2f sztuk po %.2f PLN.", assetID, additionalQuantity, newPurchasePrice)
		http.Redirect(w, r, "/", http.StatusSeeOther) // Przekieruj na stronę główną po sukcesie
		return

	} else {
		// --- Obsługa żądania GET (wyświetlanie formularza) ---
		assetID := r.URL.Query().Get("id")
		if assetID == "" {
			http.Error(w, "Brak identyfikatora aktywa do aktualizacji.", http.StatusBadRequest)
			return
		}

		portfolio, err := h.portfolioRepo.LoadPortfolio(ctx)
		if err != nil {
			http.Error(w, "Nie udało się załadować portfela.", http.StatusInternalServerError)
			log.Printf("Error loading portfolio for update asset form: %v", err)
			return
		}

		found := false
		for _, asset := range portfolio.Assets {
			if asset.ID == assetID {
				targetAsset = asset
				found = true
				break
			}
		}

		if !found {
			http.Error(w, "Aktywo nie znalezione.", http.StatusNotFound)
			return
		}

		h.renderUpdateAssetForm(w, r, targetAsset, "")
	}
}

// renderUpdateAssetForm pomaga renderować komponent UpdateAssetForm
func (h *AppHandler) renderUpdateAssetForm(w http.ResponseWriter, r *http.Request, asset models.Asset, message string) {
	err := views.UpdateAssetForm(asset, message).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Error rendering update asset form", http.StatusInternalServerError)
		log.Printf("Error rendering update asset form: %v", err)
		return
	}
}

// helper func for checking if string is in slice
func isInSlice(s string, slice []string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func (h *AppHandler) UpdateWalletTypeHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var message string           // Komunikat dla użytkownika
	var targetAsset models.Asset // Aktywo, które będziemy aktualizować/wyświetlać
	var allowedTypes = []string{"Poduszka Finansowa", "Portfel Długoterminowy", "Portfel Krótkoteminowy"}
	if r.Method == http.MethodPost {
		// --- Obsługa żądania POST (przetwarzanie formularza) ---
		err := r.ParseForm()
		if err != nil {
			message = fmt.Sprintf("Błąd parsowania formularza: %v", err)
			log.Printf("Error parsing update asset form: %v", err)
			// Spróbuj załadować aktywo, żeby formularz nie był pusty
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, a := range portfolio.Assets {
					if a.ID == r.FormValue("asset_id") {
						targetAsset = a
						break
					}
				}
			}
			h.renderUpdateWalletTypeForm(w, r, targetAsset, message)
			return
		}

		assetID := r.FormValue("asset_id")
		newWalletType := r.FormValue("new_wallet_type")

		if assetID == "" {
			message = "Brak identyfikatora aktywa do aktualizacji."
			h.renderUpdateWalletTypeForm(w, r, targetAsset, message) // targetAsset będzie puste
			return
		}

		// Sprawdź czy aktywo należy do dozwolonych typów
		if !isInSlice(newWalletType, allowedTypes) {
			// Tutaj umieść kod, który był w Twojej pętli
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, a := range portfolio.Assets {
					if a.ID == assetID {
						targetAsset = a
						break
					}
				}
			}
			h.renderUpdateWalletTypeForm(w, r, targetAsset, message)
			return
		}

		// Wywołaj funkcję repozytorium do aktualizacji aktywa
		err = h.portfolioRepo.UpdateAssetWalletType(ctx, assetID, newWalletType)
		if err != nil {
			message = fmt.Sprintf("Błąd aktualizacji aktywa: %v", err)
			log.Printf("Error updating asset (ID: %s): %v", assetID, err)
			// Spróbuj załadować aktywo, żeby formularz nie był pusty
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, a := range portfolio.Assets {
					if a.ID == assetID {
						targetAsset = a
						break
					}
				}
			}
			h.renderUpdateWalletTypeForm(w, r, targetAsset, message)
			return
		}

		log.Printf("Aktywo o ID %s zaktualizowane pomyślnie. Zmieniono typ portfela na %s.", assetID, newWalletType)
		http.Redirect(w, r, "/", http.StatusSeeOther) // Przekieruj na stronę główną po sukcesie
		return

	} else {
		// --- Obsługa żądania GET (wyświetlanie formularza) ---
		assetID := r.URL.Query().Get("id")
		if assetID == "" {
			http.Error(w, "Brak identyfikatora aktywa do aktualizacji.", http.StatusBadRequest)
			return
		}

		portfolio, err := h.portfolioRepo.LoadPortfolio(ctx)
		if err != nil {
			http.Error(w, "Nie udało się załadować portfela.", http.StatusInternalServerError)
			log.Printf("Error loading portfolio for update asset form: %v", err)
			return
		}

		found := false
		for _, asset := range portfolio.Assets {
			if asset.ID == assetID {
				targetAsset = asset
				found = true
				break
			}
		}

		if !found {
			http.Error(w, "Aktywo nie znalezione.", http.StatusNotFound)
			return
		}

		h.renderUpdateWalletTypeForm(w, r, targetAsset, "")
	}
}

// renderUpdateWalletTypeForm pomaga renderować komponent UpdateWalletTypeForm
func (h *AppHandler) renderUpdateWalletTypeForm(w http.ResponseWriter, r *http.Request, asset models.Asset, message string) {
	err := views.UpdateWalletTypeForm(asset, message).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Error rendering update asset form", http.StatusInternalServerError)
		log.Printf("Error rendering update asset form: %v", err)
		return
	}
}

// AddSubscriptionHandler obsługuje dodawanie nowych subskrypcji.
func (h *AppHandler) AddSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var message string

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			message = fmt.Sprintf("Błąd parsowania formularza: %v", err)
			log.Printf("Error parsing add subscription form: %v", err)
			h.renderAddSubscriptionForm(w, r, message)
			return
		}

		name := r.FormValue("name")
		costStr := r.FormValue("cost")
		frequency := r.FormValue("frequency")
		nextDueStr := r.FormValue("nextDue")

		cost, err := strconv.ParseFloat(costStr, 64)
		if err != nil {
			message = "Nieprawidłowa wartość 'Koszt'."
			h.renderAddSubscriptionForm(w, r, message)
			return
		}

		nextDue, err := time.Parse("2006-01-02", nextDueStr)
		if err != nil {
			message = "Nieprawidłowy format daty 'Następna Płatność'. Użyj YYYY-MM-DD."
			h.renderAddSubscriptionForm(w, r, message)
			return
		}

		newSub := models.Subscription{
			ID:        models.GenerateID(),
			Name:      name,
			Cost:      cost,
			Frequency: frequency,
			NextDue:   nextDue,
		}

		portfolio, err := h.portfolioRepo.LoadPortfolio(ctx)
		if err != nil {
			message = fmt.Sprintf("Błąd ładowania portfela: %v", err)
			log.Printf("Error loading portfolio for subscription addition: %v", err)
			h.renderAddSubscriptionForm(w, r, message)
			return
		}

		portfolio.AddSubscription(newSub)

		if err := h.portfolioRepo.SavePortfolio(ctx, portfolio); err != nil {
			message = fmt.Sprintf("Błąd zapisu portfela: %v", err)
			log.Printf("Error saving portfolio after subscription addition: %v", err)
			h.renderAddSubscriptionForm(w, r, message)
			return
		}

		log.Printf("Subscription added: %+v", newSub)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	h.renderAddSubscriptionForm(w, r, "")
}

// renderAddSubscriptionForm pomaga renderować komponent AddSubscriptionForm
func (h *AppHandler) renderAddSubscriptionForm(w http.ResponseWriter, r *http.Request, message string) {
	err := views.AddSubscriptionForm(message).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Error rendering add subscription form", http.StatusInternalServerError)
		log.Printf("Error rendering add subscription form: %v", err)
		return
	}
}

// DeleteSubscriptionHandler usuwa subskrypcję z portfela.
func (h *AppHandler) DeleteSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metoda niedozwolona", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		log.Printf("Błąd parsowania formularza POST dla usuwania subskrypcji: %v", err)
		http.Error(w, "Błąd wewnętrzny serwera", http.StatusInternalServerError)
		return
	}

	subID := r.FormValue("sub_id")
	if subID == "" {
		http.Error(w, "Brak identyfikatora subskrypcji w formularzu.", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err = h.portfolioRepo.RemoveSubscription(ctx, subID)
	if err != nil {
		log.Printf("Błąd usuwania subskrypcji (ID: %s): %v", subID, err)
		http.Error(w, fmt.Sprintf("Nie udało się usunąć subskrypcji: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Subskrypcja o ID %s usunięta pomyślnie.", subID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// UpdateSubscriptionHandler obsługuje aktualizację istniejących subskrypcji.
func (h *AppHandler) UpdateSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var message string
	var targetSub models.Subscription // Subskrypcja, którą będziemy aktualizować/wyświetlać

	if r.Method == http.MethodPost {
		// --- Obsługa żądania POST (przetwarzanie formularza) ---
		err := r.ParseForm()
		if err != nil {
			message = fmt.Sprintf("Błąd parsowania formularza: %v", err)
			log.Printf("Error parsing update subscription form: %v", err)
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, s := range portfolio.Subscriptions {
					if s.ID == r.FormValue("sub_id") {
						targetSub = s
						break
					}
				}
			}
			h.renderUpdateSubscriptionForm(w, r, targetSub, message)
			return
		}

		subID := r.FormValue("sub_id")
		name := r.FormValue("name")
		costStr := r.FormValue("cost")
		frequency := r.FormValue("frequency")
		nextDueStr := r.FormValue("nextDue")

		if subID == "" {
			message = "Brak identyfikatora subskrypcji do aktualizacji."
			h.renderUpdateSubscriptionForm(w, r, targetSub, message)
			return
		}

		cost, err := strconv.ParseFloat(costStr, 64)
		if err != nil || cost < 0 {
			message = "Nieprawidłowa wartość 'Koszt'. Musi być liczbą nieujemną."
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, s := range portfolio.Subscriptions {
					if s.ID == subID {
						targetSub = s
						break
					}
				}
			}
			h.renderUpdateSubscriptionForm(w, r, targetSub, message)
			return
		}

		nextDue, err := time.Parse("2006-01-02", nextDueStr)
		if err != nil {
			message = "Nieprawidłowy format daty 'Następna Płatność'. Użyj YYYY-MM-DD."
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, s := range portfolio.Subscriptions {
					if s.ID == subID {
						targetSub = s
						break
					}
				}
			}
			h.renderUpdateSubscriptionForm(w, r, targetSub, message)
			return
		}

		updatedSub := models.Subscription{
			ID:        subID, // Używamy istniejącego ID
			Name:      name,
			Cost:      cost,
			Frequency: frequency,
			NextDue:   nextDue,
		}

		err = h.portfolioRepo.UpdateSubscription(ctx, updatedSub)
		if err != nil {
			message = fmt.Sprintf("Błąd aktualizacji subskrypcji: %v", err)
			log.Printf("Error updating subscription (ID: %s): %v", subID, err)
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, s := range portfolio.Subscriptions {
					if s.ID == subID {
						targetSub = s
						break
					}
				}
			}
			h.renderUpdateSubscriptionForm(w, r, targetSub, message)
			return
		}

		log.Printf("Subskrypcja o ID %s zaktualizowana pomyślnie.", subID)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return

	} else {
		// --- Obsługa żądania GET (wyświetlanie formularza) ---
		subID := r.URL.Query().Get("id")
		if subID == "" {
			http.Error(w, "Brak identyfikatora subskrypcji do aktualizacji.", http.StatusBadRequest)
			return
		}

		portfolio, err := h.portfolioRepo.LoadPortfolio(ctx)
		if err != nil {
			http.Error(w, "Nie udało się załadować portfela.", http.StatusInternalServerError)
			log.Printf("Error loading portfolio for update subscription form: %v", err)
			return
		}

		found := false
		for _, sub := range portfolio.Subscriptions {
			if sub.ID == subID {
				targetSub = sub
				found = true
				break
			}
		}

		if !found {
			http.Error(w, "Subskrypcja nie znaleziona.", http.StatusNotFound)
			return
		}

		h.renderUpdateSubscriptionForm(w, r, targetSub, "")
	}
}

// renderUpdateSubscriptionForm pomaga renderować komponent UpdateSubscriptionForm
func (h *AppHandler) renderUpdateSubscriptionForm(w http.ResponseWriter, r *http.Request, sub models.Subscription, message string) {
	err := views.UpdateSubscriptionForm(sub, message).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Error rendering update subscription form", http.StatusInternalServerError)
		log.Printf("Error rendering update subscription form: %v", err)
		return
	}
}

// UpdateAssetPriceHandler obsługuje aktualizację ceny bieżącej aktywa.
func (h *AppHandler) UpdateAssetPriceHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var message string           // Komunikat dla użytkownika
	var targetAsset models.Asset // Aktywo, które będziemy aktualizować/wyświetlać

	if r.Method == http.MethodPost {
		// --- Obsługa żądania POST (przetwarzanie formularza) ---
		err := r.ParseForm()
		if err != nil {
			message = fmt.Sprintf("Błąd parsowania formularza: %v", err)
			log.Printf("Error parsing update asset form: %v", err)
			// Spróbuj załadować aktywo, żeby formularz nie był pusty
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, a := range portfolio.Assets {
					if a.ID == r.FormValue("asset_id") {
						targetAsset = a
						break
					}
				}
			}
			h.renderUpdatePriceForm(w, r, targetAsset, message)
			return
		}

		assetID := r.FormValue("asset_id")
		priceStr := r.FormValue("currentPrice")
		newPrice, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || newPrice < 0 {
			// Render form again with error
			message := "Nieprawidłowa wartość 'Nowa Cena'. Musi być liczbą nieujemną."
			portfolio, loadErr := h.portfolioRepo.LoadPortfolio(ctx)
			if loadErr == nil {
				for _, a := range portfolio.Assets {
					if a.ID == assetID {
						targetAsset = a
						break
					}
				}
			}
			h.renderUpdatePriceForm(w, r, targetAsset, message)
			return
		}

		err = h.portfolioRepo.UpdateAssetCurrentPrice(ctx, assetID, newPrice)
		if err != nil {
			log.Printf("Błąd aktualizacji ceny aktywa (ID: %s): %v", assetID, err)
			message := fmt.Sprintf("Nie udało się zaktualizować ceny: %v", err)
			h.renderUpdatePriceForm(w, r, targetAsset, message)
			return
		}

		log.Printf("Cena aktywa o ID %s zaktualizowana pomyślnie.", assetID)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else {
		// --- Obsługa żądania GET (wyświetlanie formularza) ---
		assetID := r.URL.Query().Get("id")
		if assetID == "" {
			http.Error(w, "Brak identyfikatora aktywa do aktualizacji.", http.StatusBadRequest)
			return
		}

		portfolio, err := h.portfolioRepo.LoadPortfolio(ctx)
		if err != nil {
			http.Error(w, "Nie udało się załadować portfela.", http.StatusInternalServerError)
			log.Printf("Error loading portfolio for update asset form: %v", err)
			return
		}

		found := false
		for _, asset := range portfolio.Assets {
			if asset.ID == assetID {
				targetAsset = asset
				found = true
				break
			}
		}

		if !found {
			http.Error(w, "Aktywo nie znalezione.", http.StatusNotFound)
			return
		}

		h.renderUpdatePriceForm(w, r, targetAsset, "")
	}

}

// renderUpdatePriceForm pomaga renderować komponent formularza aktualizacji ceny.
// (You will need to create a corresponding `views.UpdatePriceForm` component)
func (h *AppHandler) renderUpdatePriceForm(w http.ResponseWriter, r *http.Request, asset models.Asset, message string) {
	// This assumes you create a new view: `views.UpdatePriceForm(asset, message)`
	// For now, let's log it. You would need to create `update_price.templ`.
	log.Printf("Rendering update price form for asset: %s", asset.Name)
	//Example of what the call would look like:
	err := views.UpdatePriceForm(asset, message).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Error rendering update price form", http.StatusInternalServerError)
	}

	// Placeholder until the view is created
	//fmt.Fprintf(w, "<h1>Update Price for %s</h1><p>%s</p><form method='POST'><label>New Price:</label><input type='text' name='currentPrice' value='%.2f'><button type='submit'>Update</button></form>", asset.Name, message, asset.CurrentPrice)
}

// VisualizationsHandler - renderuje główną stronę /visualizations
func (h *AppHandler) VisualizationsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	portfolio, err := h.portfolioRepo.LoadPortfolio(ctx)
	if err != nil {
		http.Error(w, "Nie udało się załadować portfela", http.StatusInternalServerError)
		return
	}

	// Zbierz unikalne typy portfeli, aby stworzyć przyciski
	uniqueWalletTypes := make(map[string]struct{})
	for _, asset := range portfolio.Assets {
		if asset.WalletType != "" {
			uniqueWalletTypes[asset.WalletType] = struct{}{}
		}
	}

	// Zbierz unikalne typy aktywów, aby stworzyć przyciski
	uniqueAssetTypes := make(map[string]struct{})
	for _, asset := range portfolio.Assets {
		if asset.Type != "" {
			uniqueAssetTypes[asset.Type] = struct{}{}
		}
	}

	portfolioTypes := make([]string, 0, len(uniqueWalletTypes))
	for pType := range uniqueWalletTypes {
		portfolioTypes = append(portfolioTypes, pType)
	}

	assetTypes := make([]string, 0, len(uniqueAssetTypes))
	for aType := range uniqueAssetTypes {
		assetTypes = append(assetTypes, aType)
	}

	// Renderuj całą stronę
	views.VisualizationsPage(portfolioTypes, assetTypes, portfolio).Render(r.Context(), w)
}

// GetVisualizationDataHandler - renderuje i zwraca sam wykres (dla HTMX)
func (h *AppHandler) GetVisualizationDataHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	portfolio, err := h.portfolioRepo.LoadPortfolio(ctx)
	if err != nil {
		http.Error(w, "Nie udało się załadować portfela", http.StatusInternalServerError)
		return
	}

	// 1. Pobierz typ portfela z query parametru
	portfolioType := r.URL.Query().Get("portfolioType")
	log.Printf("Rendering update price form for asset: %s", portfolioType)

	// 2. Filtruj aktywa
	var filteredAssets []models.Asset
	if portfolioType == "Wszystkie" || portfolioType == "" {
		filteredAssets = portfolio.Assets
	} else {
		for _, asset := range portfolio.Assets {
			if asset.WalletType == portfolioType {
				filteredAssets = append(filteredAssets, asset)
			}
		}
	}

	// 3. Przygotuj dane dla wykresu
	// Agregujemy wartość per nazwa aktywa (np. "Bitcoin", "Akcje XYZ")
	valueByAsset := make(map[string]float64)
	for _, asset := range filteredAssets {
		valueByAsset[asset.Name] += asset.Quantity * asset.CurrentPrice
	}

	// Konwertujemy mapę na format wymagany przez go-echarts
	pieData := make([]opts.PieData, 0)
	for name, value := range valueByAsset {
		pieData = append(pieData, opts.PieData{Name: name, Value: fmt.Sprintf("%.2f", value)})
	}

	// 4. Utwórz i skonfiguruj wykres kołowy
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Skład portfela",
			Subtitle: "Podział aktywów według bieżącej wartości (" + portfolioType + ")",
		}),
		// Wyłącz legendę w opcjach globalnych, dodamy ją w serii
	)

	pie.AddSeries("Wartość", pieData).
		SetSeriesOptions(
			charts.WithPieChartOpts(opts.PieChart{
				Radius: []string{"40%", "75%"},
			}),
			charts.WithLabelOpts(opts.Label{

				Formatter: "{b}: {d}%", // Pokaż nazwę i procent
			}),
		)

	// <-- KLUCZOWA ZMIANA -->
	// 5. Zamiast pie.Render(w), konwertujemy opcje wykresu do JSON
	chartJSON := pie.JSON()

	// 6. Renderujemy nasz nowy komponent `Chart`, przekazując mu JSON
	// Używamy `views.Chart` zamiast `pie.Render`
	chartID := "portfolio-pie-chart" // Można generować losowe ID w razie potrzeby
	views.Chart(chartID, chartJSON).Render(r.Context(), w)
}
