// Package httpapi は HTTP API(/api 配下)を実装する。
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"bandmatch/internal/config"
	"bandmatch/internal/mailer"
	"bandmatch/internal/store"
)

type Server struct {
	cfg     config.Config
	pool    *pgxpool.Pool
	q       *store.Queries
	mail    mailer.Mailer
	limiter *limiter
	oauth   *oauth2.Config
}

func New(cfg config.Config, pool *pgxpool.Pool, m mailer.Mailer) *Server {
	s := &Server{cfg: cfg, pool: pool, q: store.New(pool), mail: m, limiter: newLimiter()}
	if cfg.GoogleEnabled() {
		s.oauth = &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			Endpoint:     google.Endpoint,
			RedirectURL:  cfg.FrontendURL + "/api/auth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
		}
	}
	return s
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	health := func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]bool{"ok": true}) }
	r.Get("/healthz", health)

	r.Route("/api", func(r chi.Router) {
		r.Use(s.limitBody, s.checkOrigin, s.loadUser)
		r.Get("/healthz", health)

		r.Get("/auth/config", s.handleAuthConfig)
		r.Post("/auth/signup", s.handleSignup)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Post("/auth/verify-email", s.handleVerifyEmail)
		r.Post("/auth/password/forgot", s.handleForgotPassword)
		r.Post("/auth/password/reset", s.handleResetPassword)
		r.Get("/auth/google/login", s.handleGoogleLogin)
		r.Get("/auth/google/callback", s.handleGoogleCallback)
		if s.cfg.IsDev() {
			// 開発専用。本番ではルート自体が存在しない
			r.Post("/auth/dev-login", s.handleDevLogin)
		}

		r.Get("/me", s.handleMe)
		r.Get("/posts", s.handleListPosts)
		r.Get("/posts/{id}", s.handleGetPost)
		r.Get("/users/{id}", s.handleGetUser)

		r.Group(func(r chi.Router) {
			r.Use(requireLogin)
			r.Patch("/me", s.handleUpdateMe)
			r.Get("/me/posts", s.handleMyPosts)
			r.Post("/auth/resend-verification", s.handleResendVerification)
			r.Patch("/posts/{id}", s.handleUpdatePost)
			r.Delete("/posts/{id}", s.handleDeletePost)
			r.Get("/conversations", s.handleListConversations)
			r.Get("/conversations/unread-count", s.handleUnreadCount)
			r.Get("/conversations/{id}", s.handleGetConversation)
			r.Get("/conversations/{id}/messages", s.handleListMessages)
			r.Post("/conversations/{id}/read", s.handleMarkRead)

			r.Group(func(r chi.Router) {
				// メール確認とプロフィール(年齢・性別)入力が済んだ人だけができる操作
				r.Use(requireActive)
				r.Post("/posts", s.handleCreatePost)
				r.Post("/posts/{id}/conversations", s.handleStartConversation)
				r.Post("/conversations/{id}/messages", s.handleSendMessage)
			})
		})
	})
	return r
}

// RunBackground は定期処理(送信に失敗したメールの再送、期限切れセッションの掃除)を回す。
func (s *Server) RunBackground(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.retryEmails(ctx)
			if err := s.q.DeleteExpiredSessions(ctx); err != nil && ctx.Err() == nil {
				slog.Warn("期限切れセッションの削除に失敗", "err", err)
			}
		}
	}
}

// ---- レスポンス・リクエストの共通処理 ----

type apiError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, apiError{Error: msg, Code: code})
}

func (s *Server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("内部エラー", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, 500, "internal", "サーバーでエラーが発生しました。時間をおいてもう一度お試しください。")
}

func invalid(w http.ResponseWriter, msg string) { writeError(w, 422, "validation", msg) }

func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, 413, "too_large", "送信内容が大きすぎます。")
		} else {
			writeError(w, 400, "bad_request", "リクエストの形式が正しくありません。")
		}
		return false
	}
	return true
}

func pathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		writeError(w, 404, "not_found", "見つかりませんでした。")
		return uuid.Nil, false
	}
	return id, true
}

// ---- ミドルウェア ----

func (s *Server) limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		next.ServeHTTP(w, r)
	})
}

// checkOrigin は CSRF 対策。更新系リクエストは自サイトからのものだけ受け付ける。
func (s *Server) checkOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if r.Header.Get("Origin") != s.cfg.FrontendURL {
				writeError(w, 403, "bad_origin", "不正なリクエストです。")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type ctxKey int

const userKey ctxKey = 0

func (s *Server) loadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
			u, err := s.q.GetSessionUser(r.Context(), hashToken(c.Value))
			if err == nil {
				r = r.WithContext(context.WithValue(r.Context(), userKey, &u))
			} else if !isNoRows(err) {
				s.internalError(w, r, err)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func currentUser(r *http.Request) *store.User {
	u, _ := r.Context().Value(userKey).(*store.User)
	return u
}

func requireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if currentUser(r) == nil {
			writeError(w, 401, "unauthorized", "ログインが必要です。")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func profileComplete(u *store.User) bool { return u.BirthDate != nil && u.Gender != nil }

func requireActive(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := currentUser(r)
		if u.EmailVerifiedAt == nil {
			writeError(w, 403, "email_unverified", "メールアドレスの確認が済んでいません。確認メールのリンクを開いてください。")
			return
		}
		if !profileComplete(u) {
			writeError(w, 403, "profile_incomplete", "プロフィール(生年月日・性別)を入力してください。")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	return r.RemoteAddr
}

// limiter はメモリ上の簡易レート制限(ログイン試行などの総当たり対策)。
type limiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newLimiter() *limiter { return &limiter{hits: map[string][]time.Time{}} }

func (l *limiter) allow(key string, max int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if len(l.hits) > 50000 { // メモリを使い切らないための安全弁
		l.hits = map[string][]time.Time{}
	}
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if now.Sub(t) < window {
			kept = append(kept, t)
		}
	}
	if len(kept) >= max {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}

func tooMany(w http.ResponseWriter) {
	writeError(w, 429, "rate_limited", "操作の回数が多すぎます。しばらく待ってからお試しください。")
}
