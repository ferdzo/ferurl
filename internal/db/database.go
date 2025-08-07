package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ferdzo/ferurl/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	connectionPool *pgxpool.Pool
}

type URL struct {
	ShortURL  string    `json:"shorturl"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type PageVisit struct {
	ShortURL   string    `json:"shorturl"`
	Count      int       `json:"count"`
	IP_Address string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewDatabaseClient(config utils.DatabaseConfig) (*Database, error) {
	pool, err := pgxpool.New(context.Background(), utils.DatabaseUrl())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)

		return nil, err
	}

	return &Database{connectionPool: pool}, nil
}

func (d *Database) InsertNewURL(u URL) error {
	timeNow := time.Now()
	_, err := d.connectionPool.Exec(context.Background(), "INSERT INTO urls (shorturl, url, created_at,expires_at,active) VALUES ($1, $2, $3,$4,$5)", u.ShortURL, u.URL, timeNow, u.ExpiresAt, true)
	if err != nil {
		return fmt.Errorf("failed to insert URL into database: %w", err)
	}
	return nil
}

func (d *Database) InsertAnalytics(p PageVisit) error {
	timeNow := time.Now()
	_, err := d.connectionPool.Exec(context.Background(), "INSERT INTO analytics (shorturl, count, ip_address, user_agent, created_at) VALUES ($1, $2, $3, $4, $5)", p.ShortURL, p.Count, p.IP_Address, p.UserAgent, timeNow)
	if err != nil {
		return fmt.Errorf("failed to insert analytics into database: %w", err)
	}
	return nil
}

func (d *Database) GetAnalytics(shorturl string) ([]PageVisit, error) {
	rows, err := d.connectionPool.Query(context.Background(), "SELECT shorturl, count, ip_address, user_agent, created_at FROM analytics WHERE shorturl = $1", shorturl)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve analytics from database: %w", err)
	}
	defer rows.Close()

	var analytics []PageVisit
	for rows.Next() {
		var p PageVisit
		err := rows.Scan(&p.ShortURL, &p.Count, &p.IP_Address, &p.UserAgent, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan analytics row: %w", err)
		}
		analytics = append(analytics, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over analytics rows: %w", err)
	}
	return analytics, nil
}

func (d *Database) DeleteURL(shorturl string) error {
	_, err := d.connectionPool.Exec(context.Background(), "DELETE FROM urls WHERE shorturl = $1", shorturl)
	if err != nil {
		return fmt.Errorf("failed to delete URL from database: %w", err)
	}
	return nil
}

func (d *Database) GetURL(shorturl string) (string, error) {
	var url string
	err := d.connectionPool.QueryRow(context.Background(), "SELECT url FROM urls WHERE shorturl = $1", shorturl).Scan(&url)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve URL from database: %w", err)
	}
	return url, nil
}
