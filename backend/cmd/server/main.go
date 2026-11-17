package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"bandmatch/internal/config"
	"bandmatch/internal/db"
	"bandmatch/internal/httpapi"
	"bandmatch/internal/mailer"
)

func main() {
	if err := run(); err != nil {
		slog.Error("起動に失敗しました", "err", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load() // .env があれば読む(本番は環境変数で渡す)
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	var m mailer.Mailer
	switch cfg.MailDriver {
	case "smtp":
		m = mailer.SMTP{Addr: cfg.SMTPAddr, From: cfg.MailFrom}
	case "resend":
		m = mailer.Resend{APIKey: cfg.ResendAPIKey, From: cfg.MailFrom}
	default:
		m = mailer.Log{}
	}

	srv := httpapi.New(cfg, pool, m)
	go srv.RunBackground(ctx)

	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- httpSrv.ListenAndServe() }()
	slog.Info("サーバー起動", "port", cfg.Port, "env", cfg.Env, "mail", cfg.MailDriver, "google", cfg.GoogleEnabled())

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	}
	return nil
}
