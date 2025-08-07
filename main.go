package main

import (
	"fmt"
	"net/http"

	shortner "github.com/ferdzo/ferurl/internal/shortener"
	"github.com/ferdzo/ferurl/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	log.Info("Starting ferurl URL shortener service")

	redisConfig, err := utils.LoadRedisConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to load Redis configuration")
		return
	}

	dbConfig, err := utils.LoadDbConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to load database configuration")
		return
	}

	s, err := shortner.NewService(redisConfig, dbConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize service")
		return
	}

	baseUrl := utils.GetEnv("BASE_URL", "https://url.ferdzo.xyz")
	h, err := shortner.NewHandler(s, baseUrl)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize handler")
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

	r.Post("/create", h.CreateShortURL)
	r.Get("/{key}", h.GetUrl)

	port := utils.GetEnv("PORT", "3000")
	log.Info("Starting server on port: " + port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.WithError(err).Fatal("Server failed to start")
	}
}
