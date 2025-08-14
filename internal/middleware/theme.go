package middleware

import (
	"context"
	"net/http"
)

// Definiujemy własny typ dla klucza kontekstu, aby uniknąć kolizji.
type contextKey string

const themeKey = contextKey("theme")

// ThemeMiddleware odczytuje ciasteczko "theme" i dodaje jego wartość do kontekstu żądania.
func ThemeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var theme = "light" // Domyślny motyw, jeśli ciasteczko nie istnieje

		cookie, err := r.Cookie("theme")
		if err == nil {
			// Jeśli ciasteczko istnieje i ma wartość "dark", użyj jej.
			if cookie.Value == "dark" {
				theme = "dark"
			}
		}

		// Dodaj wartość motywu do kontekstu.
		ctx := context.WithValue(r.Context(), themeKey, theme)

		// Przekaż żądanie z nowym kontekstem do następnego handlera w łańcuchu.
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Funkcja pomocnicza do łatwego odczytu motywu z kontekstu w innych częściach aplikacji.
func GetTheme(ctx context.Context) string {
	if theme, ok := ctx.Value(themeKey).(string); ok {
		return theme
	}
	return "light" // Zwróć domyślny, jeśli nic nie ma w kontekście.
}
