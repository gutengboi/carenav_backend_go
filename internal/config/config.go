package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	MongoURL         string
	MongoDBName      string
	JWTSecret        string
	Secret           string
	ResetTokenSecret string
	StreamAPIKey     string
	StreamAPISecret  string
	EmailUser        string
	EmailPass        string
	Debug            bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:             getEnv("PORT", "3000"),
		MongoURL:         os.Getenv("MONGO_URL"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		Secret:           os.Getenv("SECRET"),
		ResetTokenSecret: os.Getenv("RESET_TOKEN_SECRET"),
		StreamAPIKey:     os.Getenv("STREAM_API_KEY"),
		StreamAPISecret:  os.Getenv("STREAM_API_SECRET"),
		EmailUser:        os.Getenv("EMAIL_USER"),
		EmailPass:        os.Getenv("EMAIL_PASS"),
		Debug:            parseBool(os.Getenv("DEBUG")),
	}

	if cfg.MongoURL != "" {
		dbName, err := dbNameFromURI(cfg.MongoURL)
		if err != nil {
			return nil, fmt.Errorf("could not determine database name from MONGO_URL: %w", err)
		}
		cfg.MongoDBName = dbName
	}

	required := map[string]string{
		"MONGO_URL":          cfg.MongoURL,
		"JWT_SECRET":         cfg.JWTSecret,
		"SECRET":             cfg.Secret,
		"RESET_TOKEN_SECRET": cfg.ResetTokenSecret,
		"STREAM_API_KEY":     cfg.StreamAPIKey,
		"STREAM_API_SECRET":  cfg.StreamAPISecret,
	}
	for name, val := range required {
		if val == "" {
			return nil, fmt.Errorf("missing required environment variable %s", name)
		}
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseBool(v string) bool {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}

// dbNameFromURI mirrors what mongoose.connect(MONGO_URL) does implicitly:
// the database name is the path segment of the connection string
// (mongodb+srv://user:pass@host/dbname?params).
func dbNameFromURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return "", fmt.Errorf("no database name found in MONGO_URL path")
	}
	return name, nil
}
