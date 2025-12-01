package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Mailjet  MailjetConfig
	Xendit   XenditConfig
	Redis    RedisConfig
}

type MailjetConfig struct {
	MailjetBaseUrl           string
	MailjetBasicAuthUsername string
	MailjetBasicAuthPassword string
	MailjetSenderEmail       string
	MailjetSenderName        string
}

type AppConfig struct {
	Name                    string
	Version                 string
	Environment             string
	AppDeploymentUrl        string
	AppEmailVerificationKey string
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type JWTConfig struct {
	SecretKey string
}

type XenditConfig struct {
	XenditSecretKey                string
	XenditUrl                      string
	RedirectUrl                    string
	XenditWebhookVerificationToken string
}

type RedisConfig struct {
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		return nil, errors.New("missing redis database")
	}

	cfg := &Config{
		App: AppConfig{
			Name:                    getEnv("APP_NAME", "Futsal Booking API"),
			Version:                 getEnv("APP_VERSION", "1.0.0"),
			Environment:             getEnv("APP_ENV", "development"),
			AppDeploymentUrl:        getEnv("APP_DEPLOYMENT_URL", ""),
			AppEmailVerificationKey: getEnv("APP_EMAIL_VERIFICATION_KEY", ""),
		},
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "futsal_booking_api"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		JWT: JWTConfig{
			SecretKey: getEnv("JWT_SECRET", ""),
		},
		Mailjet: MailjetConfig{
			MailjetBaseUrl:           getEnv("MAILJET_BASE_URL", ""),
			MailjetBasicAuthUsername: getEnv("MAILJET_BASIC_AUTH_USERNAME", ""),
			MailjetBasicAuthPassword: getEnv("MAILJET_BASIC_AUTH_PASSWORD", ""),
			MailjetSenderEmail:       getEnv("MAILJET_SENDER_EMAIL", ""),
			MailjetSenderName:        getEnv("MAILJET_SENDER_NAME", ""),
		},
		Xendit: XenditConfig{
			XenditSecretKey:                getEnv("XENDIT_SECRET_KEY", ""),
			XenditUrl:                      getEnv("XENDIT_URL", ""),
			RedirectUrl:                    getEnv("REDIRECT_URL", ""),
			XenditWebhookVerificationToken: getEnv("XENDIT_WEBHOOK_VERIFICATION_TOKEN", ""),
		},
		Redis: RedisConfig{
			RedisHost:     getEnv("REDIS_HOST", "localhost"),
			RedisPort:     getEnv("REDIS_PORT", "6379"),
			RedisPassword: getEnv("REDIS_PASSWORD", ""),
			RedisDB:       redisDB,
		},
	}

	if cfg.JWT.SecretKey == "" {
		return nil, errors.New("missing jwt secret")
	}

	if cfg.App.AppDeploymentUrl == "" {
		return nil, errors.New("missing app deployment url")
	}

	if cfg.App.AppEmailVerificationKey == "" {
		return nil, errors.New("missing app email verification key")
	}

	if cfg.Database.Password == "" {
		return nil, errors.New("missing database password")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return defaultVal
}
