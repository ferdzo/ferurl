package shortener

import (
	"fmt"
	"net/http"

	"github.com/ferdzo/ferurl/internal/cache"
	"github.com/ferdzo/ferurl/internal/db"

	"github.com/ferdzo/ferurl/utils"
	"github.com/go-chi/chi/v5"
)

type Service struct {
	cache    *cache.Cache
	database *db.Database
}

type Handler struct {
	service *Service
}

func NewService(redisConfig utils.RedisConfig, databaseConfig utils.DatabaseConfig) (*Service, error) {
	redisClient, err := cache.NewRedisClient(redisConfig)
	if err != nil {
		return nil, err
	}
	databaseClient, err := db.NewDatabaseClient(databaseConfig)
	if err != nil {
		return nil, err
	}

	return &Service{cache: redisClient, database: databaseClient}, nil
}

func NewHandler(service *Service) (*Handler, error) {
	if service == nil {
		return nil, fmt.Errorf("service is nil")
	}

	return &Handler{service: service}, nil
}

func (h *Handler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	longUrl := r.FormValue("url")

	if longUrl == "" {
		http.Error(w, "Long URL is required", http.StatusBadRequest)
		return
	}
	if !utils.IsValidUrl(longUrl) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	shortUrl := generateShortUrl(longUrl)

	http.Redirect(w, r, shortUrl, http.StatusSeeOther)
}

func (h *Handler) GetUrl(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "key")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	shortUrl, err := h.fetchUrl(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, shortUrl, http.StatusSeeOther)
}

func generateShortUrl(url string) string {
	shortUrl := utils.GenerateUrlHash(url)[:7]

	return shortUrl
}

func (h *Handler) fetchUrl(id string) (string, error) {
	if url, err := h.service.fetchUrlFromCache(id); err == nil {
		return url, nil
	}
	if url, err := h.service.fetchUrlFromDatabase(id); err == nil {
		return url, nil
	}
	return "", fmt.Errorf("URL not found")
}

func (s *Service) fetchUrlFromCache(shortUrl string) (string, error) {
	if url, err := s.cache.Get(shortUrl); err == nil {
		return url, nil
	}
	return "", fmt.Errorf("URL not found")
}

func (s *Service) fetchUrlFromDatabase(id string) (string, error) {
	if url, err := s.database.GetURL(id); err == nil {
		return url, nil
	}
	return "", fmt.Errorf("URL not found")
}
