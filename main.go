package main

import (
	"fmt"
	"net/http"

	shortner "github.com/ferdzo/ferurl/internal/shortener"
	"github.com/ferdzo/ferurl/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	fmt.Println("Welcome to ferurl, a simple URL shortener!")

	redisConfig, err := utils.LoadRedisConfig()
	if err != nil {
		fmt.Println("Error loading Redis config:", err)
		return
	}
	dbConfig, err := utils.LoadDbConfig()
	if err != nil {
		fmt.Println("Error loading DB config:", err)
		return
	}

	s, err := shortner.NewService(redisConfig, dbConfig)
	if err != nil {
		fmt.Println("Error creating service:", err)
		return
	}

	h, err := shortner.NewHandler(s)
	if err != nil {
		fmt.Println("Error creating handler:", err)
		return
	}
	initServer(*h)
}

func initServer(h shortner.Handler) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Welcome to ferurl!")
	})
	fmt.Println("Server started on port 3000")
	r.Post("/create", h.CreateShortURL)
	r.Get("/{key}", h.GetUrl)
	http.FileServer(http.Dir("web"))

	http.ListenAndServe(":3000", r)
}
