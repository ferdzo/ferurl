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

	redisConfig := utils.LoadRedisConfig()
	dbConfig := utils.LoadDbConfig()

	s, err := shortner.NewService(redisConfig, dbConfig)
	if err != nil {
		fmt.Println("Error creating service:", err)
		return
	}
	baseUrl := utils.GetEnv("BASE_URL", "https://url.ferdzo.xyz")
	h, err := shortner.NewHandler(s, baseUrl)
	if err != nil {
		fmt.Println("Error creating handler:", err)
		return
	}

	initServer(*h)
}

func initServer(h shortner.Handler) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/index.html")
	})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Welcome to ferurl!")
	})
	fmt.Println("Server started on port 3000")
	r.Post("/create", h.CreateShortURL)
	r.Get("/{key}", h.GetUrl)

	http.ListenAndServe(":3000", r)
}
