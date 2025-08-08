package shortener

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ferdzo/ferurl/internal/cache"
	"github.com/ferdzo/ferurl/internal/db"
	"github.com/ferdzo/ferurl/utils"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

type Service struct {
	cache    *cache.Cache
	database *db.Database
}

type Handler struct {
	service *Service
	baseURL string
}

func NewService(redisConfig utils.RedisConfig, databaseConfig utils.DatabaseConfig) (*Service, error) {
	log.Info("Initializing Redis client")
	redisClient, err := cache.NewRedisClient(redisConfig)
	if err != nil {
		log.WithError(err).Error("Failed to initialize Redis client")
		return nil, err
	}

	log.Info("Initializing Database client")
	databaseClient, err := db.NewDatabaseClient(databaseConfig)
	if err != nil {
		log.WithError(err).Error("Failed to initialize Database client")
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
		log.WithError(err).Warn("Invalid JSON input received")
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}

	longUrl := input.URL
	if longUrl == "" {
		log.Warn("Empty URL received")
		http.Error(w, "Long URL is required", http.StatusBadRequest)
		return
	}
	if !utils.IsValidUrl(longUrl) {
		log.WithField("url", longUrl).Warn("Invalid URL format received")
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	expiresAt := input.ExpiresAt

	shortUrl := generateShortUrl(longUrl)
	fetchedShortUrl, err := h.fetchUrl(shortUrl)
	if err == nil && fetchedShortUrl != "" {
		fullShortUrl := fmt.Sprintf("%s%s", h.baseURL, shortUrl)
		json.NewEncoder(w).Encode(map[string]string{"short_url": fullShortUrl})
		return
	}
	if err != nil && isNotFoundError(err) {
		log.WithField("short_url", shortUrl).Debug("Creating new short URL")
	}

	newUrl := db.URL{
		URL:       longUrl,
		ShortURL:  shortUrl,
		ExpiresAt: expiresAt,
	}

	err = h.service.storeUrl(newUrl)
	if err != nil {
		log.Error("Failed to store URL", "error", err)
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
		log.Warn("Empty short URL key received")
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	if !utils.IsValidShortUrl(shortKey) {
		log.WithField("key", shortKey).Warn("Invalid short URL format received")
		http.Error(w, "Invalid short URL", http.StatusBadRequest)
		return
	}

	longUrl, err := h.fetchUrl(shortKey)
	if err != nil {
		if isNotFoundError(err) {
			http.Error(w, "URL not found", http.StatusNotFound)
		} else {
			log.WithFields(logrus.Fields{
				"short_url": shortKey,
				"error":     err.Error(),
			}).Error("Failed to fetch URL")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	pv := db.PageVisit{
		ShortURL:   shortKey,
		Count:      1,
		IP_Address: r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		CreatedAt:  time.Now(),
	}

	go func(pageVisit db.PageVisit) {
		defer func() {
			if r := recover(); r != nil {
				log.WithField("error", r).Error("Panic in analytics insertion")
			}
		}()
		if err := h.service.database.InsertAnalytics(pageVisit); err != nil {
			logrus.Error("Failed to store analytics", "error", err)
		}
	}(pv)

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		url, err := h.service.fetchUrlFromCache(id)
		select {
		case resultChan <- result{url, err, "cache"}:
		case <-ctx.Done():
			return
		}
	}()

	go func() {
		defer wg.Done()
		url, err := h.service.fetchUrlFromDatabase(id)
		select {
		case resultChan <- result{url, err, "db"}:
		case <-ctx.Done():
			return
		}
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var foundInDB string
	var cacheHit bool
	var lastErr error

	for range 2 {
		select {
		case res, ok := <-resultChan:
			if !ok {
				break
			}
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
		case <-ctx.Done():
			return "", fmt.Errorf("timeout while fetching URL: %w", ctx.Err())
		}
	}

	if foundInDB != "" {
		go func(cacheKey, url string) {
			defer func() {
				if r := recover(); r != nil {
					log.Error("Panic in cache update", "error", r)
				}
			}()
			if err := h.service.cache.Set(cacheKey, url); err != nil {
				log.Error("Failed to update URL in cache", "error", err)
			}
		}(id, foundInDB)

		return foundInDB, nil
	}

	if isNotFoundError(lastErr) {
		log.Warn("URL not found")
		return "", nil
	}
	return "", lastErr
}

func (s *Service) fetchUrlFromCache(shortUrl string) (string, error) {
	url, err := s.cache.Get(shortUrl)
	if err != nil {
		log.WithFields(logrus.Fields{
			"short_url": shortUrl,
			"error":     err,
		}).Debug("Cache miss")
		return "", err
	}
	return url, nil

}

func (s *Service) fetchUrlFromDatabase(shortUrl string) (string, error) {
	url, err := s.database.GetURL(shortUrl)
	if err != nil {
		log.WithFields(logrus.Fields{
			"short_url": shortUrl,
			"error":     err,
		}).Debug("Database lookup failed")
		return "", err
	}
	log.WithField("short_url", shortUrl).Debug("Database hit")
	return url, nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "redis: nil") ||
		strings.Contains(errMsg, "no rows in result set")
}

func (s *Service) storeUrl(u db.URL) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := s.cache.Set(u.ShortURL, u.URL); err != nil {
			log.WithFields(logrus.Fields{
				"short_url": u.ShortURL,
				"error":     err,
			}).Error("Failed to store URL in cache")
			select {
			case errChan <- err:
			default:
			}
		} else {
			log.WithField("short_url", u.ShortURL).Debug("Stored URL in cache")
		}
	}()

	go func() {
		defer wg.Done()
		if err := s.database.InsertNewURL(u); err != nil {
			log.WithFields(logrus.Fields{
				"short_url": u.ShortURL,
				"error":     err,
			}).Error("Failed to store URL in database")
			select {
			case errChan <- err:
			default:
			}
		} else {
			log.WithField("short_url", u.ShortURL).Debug("Stored URL in database")
		}
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}
