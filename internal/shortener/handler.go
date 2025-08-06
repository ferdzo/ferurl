package shortener

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

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
	baseURL string
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

func NewHandler(service *Service, baseURL string) (*Handler, error) {
	if service == nil {
		return nil, fmt.Errorf("service is nil")
	}

	return &Handler{service: service, baseURL: baseURL}, nil
}
func (h *Handler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var input struct {
		URL       string    `json:"url"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}

	longUrl := input.URL
	if longUrl == "" {
		http.Error(w, "Long URL is required", http.StatusBadRequest)
		return
	}
	if !utils.IsValidUrl(longUrl) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	expiresAt := input.ExpiresAt

	shortUrl := generateShortUrl(longUrl)
	fetchedShortUrl, _ := h.fetchUrl(shortUrl)
	if fetchedShortUrl != "" {
		fullShortUrl := fmt.Sprintf("%s%s", h.baseURL, shortUrl)
		json.NewEncoder(w).Encode(map[string]string{"short_url": fullShortUrl})
		return
	}

	newUrl := db.URL{
		URL:       longUrl,
		ShortURL:  shortUrl,
		ExpiresAt: expiresAt,
	}

	if err := h.service.storeUrl(newUrl); err != nil {
		fmt.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	fullShortUrl := fmt.Sprintf("%s%s", h.baseURL, shortUrl)
	json.NewEncoder(w).Encode(map[string]string{"short_url": fullShortUrl})

}

func (h *Handler) GetUrl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	shortKey := chi.URLParam(r, "key")
	if shortKey == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	if !utils.IsValidShortUrl(shortKey) {
		http.Error(w, "Invalid short URL", http.StatusBadRequest)
		return
	}

	longUrl, err := h.fetchUrl(shortKey)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "URL not found", http.StatusInternalServerError)
		return
	}

	pv := db.PageVisit{
		ShortURL:   shortKey,
		Count:      1,
		IP_Address: r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		CreatedAt:  time.Now(),
	}

	err = h.service.database.InsertAnalytics(pv)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, longUrl, http.StatusSeeOther)
}

func generateShortUrl(url string) string {
	shortUrl := utils.GenerateUrlHash(url)[:7]

	return shortUrl
}

func (h *Handler) fetchUrl(id string) (string, error) {
	type result struct {
		url    string
		err    error
		source string
	}
	resultChan := make(chan result, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		url, err := h.service.fetchUrlFromCache(id)
		select {
		case resultChan <- result{url, err, "cache"}:
		case <-ctx.Done():
			return
		}
	}()

	go func() {
		url, err := h.service.fetchUrlFromDatabase(id)
		select {
		case resultChan <- result{url, err, "db"}:
		case <-ctx.Done():
			return
		}
	}()

	var foundInDB string
	var cacheHit bool
	var lastErr error

	for range 2 {
		res := <-resultChan
		if res.err == nil {
			if res.source == "db" {
				foundInDB = res.url
				if !cacheHit {
					continue
				}
			} else if res.source == "cache" {
				cacheHit = true
				if foundInDB != "" {
					return res.url, nil
				}
			}

			return res.url, nil
		}
		lastErr = res.err
	}

	if foundInDB != "" {
		go func() {
			_ = h.service.cache.Set(id, foundInDB)
			fmt.Println("URL updated in Redis")
		}()
		return foundInDB, nil
	}

	return "", lastErr
}

func (s *Service) fetchUrlFromCache(shortUrl string) (string, error) {
	if url, err := s.cache.Get(shortUrl); err == nil {
		return url, nil
	}
	return "", fmt.Errorf("URL not found")
}

func (s *Service) fetchUrlFromDatabase(shortUrl string) (string, error) {
	if url, err := s.database.GetURL(shortUrl); err == nil {
		return url, nil
	}

	return "", fmt.Errorf("URL not found")
}

func (s *Service) storeUrl(u db.URL) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.cache.Set(u.ShortURL, u.URL); err != nil {
			errChan <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.database.InsertNewURL(u); err != nil {
			errChan <- err
		}
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		return err
	}

	return nil
}
