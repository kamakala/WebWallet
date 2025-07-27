package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"webwallet/internal/models" // Importujemy nasze modele
)

// RepoConfiguration przechowuje konfigurację połączenia z bazą danych
type RepoConfiguration struct {
	URI        string
	Database   string
	Collection string
}

// PortfolioRepo implementuje operacje CRUD dla InvestmentPortfolio.
type PortfolioRepo struct {
	client     *mongo.Client
	collection *mongo.Collection
}

// NewPortfolioRepo tworzy nową instancję PortfolioRepo i łączy się z MongoDB.
func NewPortfolioRepo(config RepoConfiguration) (*PortfolioRepo, error) {
	// Kontekst z timeoutem na połączenie
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // Upewnij się, że kontekst zostanie anulowany po zakończeniu funkcji

	// Opcje połączenia
	clientOptions := options.Client().ApplyURI(config.URI)

	// Łączenie z MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Pingowanie bazy danych, aby sprawdzić połączenie
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	log.Println("Connected to MongoDB!")

	collection := client.Database(config.Database).Collection(config.Collection)

	return &PortfolioRepo{
		client:     client,
		collection: collection,
	}, nil
}

// Disconnect zamyka połączenie z bazą danych.
func (r *PortfolioRepo) Disconnect(ctx context.Context) error {
	if r.client == nil {
		return nil
	}
	return r.client.Disconnect(ctx)
}

// SavePortfolio zapisuje lub aktualizuje portfel w bazie danych.
// Na razie zakładamy, że mamy tylko jeden portfel (lub będziemy używać stałego ID).
func (r *PortfolioRepo) SavePortfolio(ctx context.Context, portfolio *models.InvestmentPortfolio) error {
	// Przykład: zawsze zapisujemy portfel z ID "main_portfolio"
	// W prawdziwej aplikacji ID portfela byłoby związane z użytkownikiem
	filter := bson.M{"_id": "main_portfolio"}

	// Opcje aktualizacji: Upsert = true oznacza "wstaw, jeśli nie istnieje"
	opts := options.Update().SetUpsert(true)

	// Aktualizacja dokumentu
	// Używamy $set, aby zaktualizować tylko podane pola, nie nadpisując całego dokumentu
	// (choć w tym przypadku nadpisanie całego dokumentu też byłoby ok).
	// Zauważ, że InvestmentPortfolio jest zapisywany bezpośrednio, MongoDB zajmie się mapowaniem.
	_, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": portfolio}, opts)
	if err != nil {
		return fmt.Errorf("failed to save portfolio: %w", err)
	}
	return nil
}

// LoadPortfolio ładuje portfel z bazy danych.
func (r *PortfolioRepo) LoadPortfolio(ctx context.Context) (*models.InvestmentPortfolio, error) {
	var portfolio models.InvestmentPortfolio
	filter := bson.M{"_id": "main_portfolio"} // Ładujemy portfel o stałym ID

	err := r.collection.FindOne(ctx, filter).Decode(&portfolio)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("No existing portfolio found. Creating a new one.")
			return models.NewInvestmentPortfolio(), nil // Zwróć nowy, pusty portfel
		}
		return nil, fmt.Errorf("failed to load portfolio: %w", err)
	}
	return &portfolio, nil
}

// RemoveAsset usuwa aktywo o podanym ID z portfela w bazie danych.
func (r *PortfolioRepo) RemoveAsset(ctx context.Context, assetID string) error {
	// 1. Załaduj aktualny portfel
	portfolio, err := r.LoadPortfolio(ctx)
	if err != nil {
		return fmt.Errorf("failed to load portfolio for asset removal: %w", err)
	}

	// 2. Znajdź i usuń aktywo z tablicy w pamięci
	found := false
	var updatedAssets []models.Asset
	for _, asset := range portfolio.Assets {
		if asset.ID == assetID {
			found = true
			log.Printf("Found asset to remove: %s", asset.Name)
		} else {
			updatedAssets = append(updatedAssets, asset)
		}
	}

	if !found {
		return fmt.Errorf("asset with ID %s not found in portfolio", assetID)
	}

	portfolio.Assets = updatedAssets
	portfolio.CalculateTotals() // Przelicz wartości portfela po usunięciu

	// 3. Zapisz zaktualizowany portfel z powrotem do bazy danych
	// Używamy SavePortfolio, które jest upsertem i zaktualizuje cały dokument portfela.
	if err := r.SavePortfolio(ctx, portfolio); err != nil {
		return fmt.Errorf("failed to save portfolio after asset removal: %w", err)
	}

	log.Printf("Asset with ID %s successfully removed from portfolio.", assetID)
	return nil
}

// UpdateAsset aktualizuje aktywo o podanym ID, dodając nową ilość i przeliczając średni koszt zakupu.
func (r *PortfolioRepo) UpdateAsset(ctx context.Context, assetID string, additionalQuantity, newPurchasePrice float64) error {
	// 1. Załaduj aktualny portfel
	portfolio, err := r.LoadPortfolio(ctx)
	if err != nil {
		return fmt.Errorf("failed to load portfolio for asset update: %w", err)
	}

	// 2. Znajdź aktywo do zaktualizowania
	found := false
	for i, asset := range portfolio.Assets {
		if asset.ID == assetID {
			found = true

			// Przelicz nową średnią cenę zakupu
			// Nowy_AvgCost = ((Stara_Ilosc * Stary_AvgCost) + (Nowa_Ilosc * Nowa_Cena_Zakupu)) / (Stara_Ilosc + Nowa_Ilosc)
			oldTotalCost := asset.Quantity * asset.AvgCost
			newTotalCostForAdditional := additionalQuantity * newPurchasePrice

			newQuantity := asset.Quantity + additionalQuantity

			// Unikaj dzielenia przez zero, choć przy quantity > 0 i additionalQuantity > 0 nie powinno się zdarzyć
			if newQuantity == 0 {
				return fmt.Errorf("cannot update asset: total quantity would be zero")
			}

			newAvgCost := (oldTotalCost + newTotalCostForAdditional) / newQuantity

			// Zaktualizuj aktywo w pamięci
			portfolio.Assets[i].Quantity = newQuantity
			portfolio.Assets[i].AvgCost = newAvgCost
			//portfolio.Assets[i].CurrentPrice = newAvgCost // Na razie CurrentPrice = AvgCost

			log.Printf("Asset %s updated: New Quantity=%.2f, New AvgCost=%.2f", asset.Name, newQuantity, newAvgCost)
			break
		}
	}

	if !found {
		return fmt.Errorf("asset with ID %s not found in portfolio for update", assetID)
	}

	portfolio.CalculateTotals() // Przelicz wartości portfela po aktualizacji

	// 3. Zapisz zaktualizowany portfel z powrotem do bazy danych
	if err := r.SavePortfolio(ctx, portfolio); err != nil {
		return fmt.Errorf("failed to save portfolio after asset update: %w", err)
	}

	return nil
}

// RemoveSubscription usuwa subskrypcję o podanym ID z portfela w bazie danych.
func (r *PortfolioRepo) RemoveSubscription(ctx context.Context, subID string) error {
	portfolio, err := r.LoadPortfolio(ctx)
	if err != nil {
		return fmt.Errorf("failed to load portfolio for subscription removal: %w", err)
	}

	found := false
	var updatedSubscriptions []models.Subscription
	for _, sub := range portfolio.Subscriptions {
		if sub.ID == subID {
			found = true
			log.Printf("Found subscription to remove: %s", sub.Name)
		} else {
			updatedSubscriptions = append(updatedSubscriptions, sub)
		}
	}

	if !found {
		return fmt.Errorf("subscription with ID %s not found in portfolio", subID)
	}

	portfolio.Subscriptions = updatedSubscriptions
	portfolio.CalculateTotals() // Przelicz wartości portfela po usunięciu

	if err := r.SavePortfolio(ctx, portfolio); err != nil {
		return fmt.Errorf("failed to save portfolio after subscription removal: %w", err)
	}

	log.Printf("Subscription with ID %s successfully removed from portfolio.", subID)
	return nil
}

// UpdateSubscription aktualizuje subskrypcję o podanym ID.
func (r *PortfolioRepo) UpdateSubscription(ctx context.Context, updatedSub models.Subscription) error {
	portfolio, err := r.LoadPortfolio(ctx)
	if err != nil {
		return fmt.Errorf("failed to load portfolio for subscription update: %w", err)
	}

	found := false
	for i, sub := range portfolio.Subscriptions {
		if sub.ID == updatedSub.ID {
			// Zastąp starą subskrypcję nową
			portfolio.Subscriptions[i] = updatedSub
			found = true
			log.Printf("Subscription %s updated.", updatedSub.Name)
			break
		}
	}

	if !found {
		return fmt.Errorf("subscription with ID %s not found in portfolio for update", updatedSub.ID)
	}

	portfolio.CalculateTotals() // Przelicz wartości portfela po aktualizacji

	if err := r.SavePortfolio(ctx, portfolio); err != nil {
		return fmt.Errorf("failed to save portfolio after subscription update: %w", err)
	}

	return nil
}
