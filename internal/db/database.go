package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ferdzo/ferurl/utils"
	"github.com/jackc/pgx/v5"
)

type Database struct {
	client *pgx.Conn
}

func NewDatabaseClient(config utils.DatabaseConfig) (*Database, error) {
	conn, err := pgx.Connect(context.Background(), utils.DatabaseUrl())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return nil, err
	}

	return &Database{client: conn}, nil
}

func (d *Database) InsertNewURL(shorturl string, url string) error {
	timeNow := time.Now()
	_, err := d.client.Exec(context.Background(), "INSERT INTO urls (shorturl, url, created_at) VALUES ($1, $2, $3)", shorturl, url, timeNow)
	if err != nil {
		return fmt.Errorf("failed to insert URL into database: %w", err)
	}
	return nil
}

func (d *Database) DeleteURL(shorturl string) error {
	_, err := d.client.Exec(context.Background(), "DELETE FROM urls WHERE shorturl = $1", shorturl)
	if err != nil {
		return fmt.Errorf("failed to delete URL from database: %w", err)
	}
	return nil
}

func (d *Database) GetURL(shorturl string) (string, error) {
	var url string
	err := d.client.QueryRow(context.Background(), "SELECT url FROM urls WHERE shorturl = $1", shorturl).Scan(&url)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve URL from database: %w", err)
	}
	return url, nil
}
