package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

// PostgresDB represents a PostgreSQL database connection
type PostgresDB struct {
	*sql.DB
	logger zerolog.Logger
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg DatabaseConfig) (*PostgresDB, error) {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Override with environment variables if they exist
	if host := getEnv("DB_HOST", ""); host != "" {
		cfg.Host = host
	}
	if user := getEnv("DB_USER", ""); user != "" {
		cfg.User = user
	}
	if password := getEnv("DB_PASSWORD", ""); password != "" {
		cfg.Password = password
	}
	if name := getEnv("DB_NAME", ""); name != "" {
		cfg.Name = name
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: log.Writer()}).With().Timestamp().Logger()
	logger.Info().Msg("Connected to PostgreSQL database")

	return &PostgresDB{
		DB:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (db *PostgresDB) Close() error {
	return db.DB.Close()
}

// GetDB returns the underlying sql.DB
func (db *PostgresDB) GetDB() *sql.DB {
	return db.DB
}

// getEnv returns the value of the environment variable or the default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}