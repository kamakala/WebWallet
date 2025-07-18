package handlers

import (
	"log"
	"net/http"
	"text/template"
)

type PageData struct {
	Title   string
	Content string
}

var tmpl *template.Template

func init() {
	var err error
	tmpl, err = template.ParseFiles(
		"internal/templates/layout.html",
		"internal/templates/home.html",
	)
	if err != nil {
		log.Fatalf("Error parsing templates: %v", err)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:   "Strona Główna",
		Content: "Witaj w Twoim porfelu",
	}
	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	// szablon render
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error executing template: %v", err)
		return
	}
}
