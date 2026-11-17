// Package config は環境変数から設定を読み込む。
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Env         string // development | production
	Port        string
	DatabaseURL string
	// FrontendURL はブラウザから見えるサイトの URL。API もこのドメインの /api 経由で呼ばれる。
	FrontendURL string

	GoogleClientID     string
	GoogleClientSecret string

	MailDriver   string // smtp | resend | log
	MailFrom     string
	SMTPAddr     string
	ResendAPIKey string
}

func (c Config) IsDev() bool         { return c.Env == "development" }
func (c Config) GoogleEnabled() bool { return c.GoogleClientID != "" && c.GoogleClientSecret != "" }
func (c Config) SecureCookies() bool { return strings.HasPrefix(c.FrontendURL, "https://") }

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() (Config, error) {
	c := Config{
		Env:                getenv("APP_ENV", "development"),
		Port:               getenv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		FrontendURL:        strings.TrimRight(getenv("FRONTEND_URL", "http://localhost:3000"), "/"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		MailDriver:         getenv("MAIL_DRIVER", "log"),
		MailFrom:           getenv("MAIL_FROM", "BandMatch <no-reply@localhost>"),
		SMTPAddr:           getenv("SMTP_ADDR", "localhost:1025"),
		ResendAPIKey:       os.Getenv("RESEND_API_KEY"),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL が設定されていません")
	}
	if c.Env != "development" && c.Env != "production" {
		return c, fmt.Errorf("APP_ENV は development か production を指定してください: %q", c.Env)
	}
	u, err := url.Parse(c.FrontendURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return c, fmt.Errorf("FRONTEND_URL が不正です: %q", c.FrontendURL)
	}
	if c.Env == "production" && !c.SecureCookies() {
		return c, fmt.Errorf("本番では FRONTEND_URL を https にしてください")
	}
	if c.MailDriver == "resend" && c.ResendAPIKey == "" {
		return c, fmt.Errorf("MAIL_DRIVER=resend には RESEND_API_KEY が必要です")
	}
	return c, nil
}
