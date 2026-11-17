package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"

	"bandmatch/internal/store"
)

const (
	sessionCookie  = "bm_session"
	stateCookie    = "bm_oauth_state"
	verifierCookie = "bm_oauth_verifier"

	sessionTTL     = 30 * 24 * time.Hour
	verifyTokenTTL = 48 * time.Hour
	resetTokenTTL  = time.Hour
)

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// DB にはトークンそのものではなくハッシュだけを保存する。
func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

func normalizeEmail(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) > 254 {
		return "", false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s || !strings.Contains(s[strings.LastIndex(s, "@")+1:], ".") {
		return "", false
	}
	return s, true
}

func validPassword(p string) string {
	if len(p) < 8 {
		return "パスワードは8文字以上にしてください。"
	}
	if len(p) > 72 {
		return "パスワードは72文字以内にしてください。"
	}
	return ""
}

func (s *Server) setCookie(w http.ResponseWriter, name, value string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: value, Path: "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true, Secure: s.cfg.SecureCookies(), SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: s.cfg.SecureCookies(), SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) startSession(ctx context.Context, w http.ResponseWriter, userID uuid.UUID) error {
	token := newToken()
	err := s.q.CreateSession(ctx, store.CreateSessionParams{
		TokenHash: hashToken(token), UserID: userID, ExpiresAt: time.Now().Add(sessionTTL),
	})
	if err != nil {
		return err
	}
	s.setCookie(w, sessionCookie, token, sessionTTL)
	return nil
}

func (s *Server) sendVerification(ctx context.Context, u store.User) error {
	if err := s.q.DeleteUserEmailTokens(ctx, store.DeleteUserEmailTokensParams{UserID: u.ID, Kind: "verify"}); err != nil {
		return err
	}
	token := newToken()
	err := s.q.CreateEmailToken(ctx, store.CreateEmailTokenParams{
		TokenHash: hashToken(token), UserID: u.ID, Kind: "verify", ExpiresAt: time.Now().Add(verifyTokenTTL),
	})
	if err != nil {
		return err
	}
	link := s.cfg.FrontendURL + "/verify-email?token=" + url.QueryEscape(token)
	s.sendEmail(ctx, u.Email, "【BandMatch】メールアドレスの確認",
		fmt.Sprintf("%s さん\n\nBandMatch へのご登録ありがとうございます。\n下のリンクを開いて、メールアドレスの確認を完了してください(48時間有効)。\n\n%s", u.DisplayName, link)+mailFooter)
	return nil
}

// ---- ハンドラー ----

func (s *Server) handleAuthConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]bool{"google_enabled": s.oauth != nil, "dev_login": s.cfg.IsDev()})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		writeJSON(w, 200, map[string]any{"user": nil})
		return
	}
	writeJSON(w, 200, map[string]any{"user": meJSON(u)})
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
		BirthDate   string `json:"birth_date"`
		Gender      string `json:"gender"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if !s.limiter.allow("signup:"+clientIP(r), 10, time.Hour) {
		tooMany(w)
		return
	}
	email, ok := normalizeEmail(in.Email)
	if !ok {
		invalid(w, "メールアドレスを正しく入力してください。")
		return
	}
	if msg := validPassword(in.Password); msg != "" {
		invalid(w, msg)
		return
	}
	name := cleanText(in.DisplayName)
	if name == "" || runeLen(name) > 30 {
		invalid(w, "表示名は1〜30文字で入力してください。")
		return
	}
	birth, msg := parseBirthDate(in.BirthDate)
	if msg != "" {
		invalid(w, msg)
		return
	}
	if !contains(genderOptions, in.Gender) {
		invalid(w, "性別を選択してください。")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	hashStr := string(hash)
	u, err := s.q.CreateUser(r.Context(), store.CreateUserParams{
		Email: email, PasswordHash: &hashStr, DisplayName: name, BirthDate: &birth, Gender: &in.Gender,
	})
	if isUniqueViolation(err) {
		writeError(w, 409, "email_taken", "このメールアドレスは既に登録されています。ログインするか、パスワードの再設定をお試しください。")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if err := s.sendVerification(r.Context(), u); err != nil {
		slog.Error("確認メールの準備に失敗", "err", err)
	}
	if err := s.startSession(r.Context(), w, u.ID); err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, 201, map[string]any{"user": meJSON(&u)})
}

// 存在しないアドレスでも応答時間が変わらないようにするためのダミー
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if !s.limiter.allow("login-ip:"+clientIP(r), 30, 15*time.Minute) ||
		!s.limiter.allow("login-email:"+email, 10, 15*time.Minute) {
		tooMany(w)
		return
	}
	fail := func() {
		writeError(w, 401, "invalid_credentials", "メールアドレスまたはパスワードが正しくありません。")
	}
	u, err := s.q.GetUserByEmail(r.Context(), email)
	if err != nil && !isNoRows(err) {
		s.internalError(w, r, err)
		return
	}
	if err != nil || u.PasswordHash == nil {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(in.Password))
		fail()
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(in.Password)) != nil {
		fail()
		return
	}
	if err := s.startSession(r.Context(), w, u.ID); err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"user": meJSON(&u)})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if err := s.q.DeleteSession(r.Context(), hashToken(c.Value)); err != nil {
			s.internalError(w, r, err)
			return
		}
	}
	s.clearCookie(w, sessionCookie)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	userID, err := s.q.ConsumeEmailToken(r.Context(), store.ConsumeEmailTokenParams{TokenHash: hashToken(in.Token), Kind: "verify"})
	if isNoRows(err) {
		writeError(w, 400, "invalid_token", "このリンクは無効か、有効期限が切れています。確認メールを再送してください。")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if err := s.q.MarkEmailVerified(r.Context(), userID); err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleResendVerification(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u.EmailVerifiedAt != nil {
		writeJSON(w, 200, map[string]bool{"ok": true})
		return
	}
	if !s.limiter.allow("resend:"+u.ID.String(), 3, 15*time.Minute) {
		tooMany(w)
		return
	}
	if err := s.sendVerification(r.Context(), *u); err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if !s.limiter.allow("forgot-ip:"+clientIP(r), 10, time.Hour) {
		tooMany(w)
		return
	}
	// 登録の有無が外から分からないよう、結果にかかわらず同じ応答を返す
	ok := func() { writeJSON(w, 200, map[string]bool{"ok": true}) }
	if !s.limiter.allow("forgot-email:"+email, 3, time.Hour) {
		ok()
		return
	}
	u, err := s.q.GetUserByEmail(r.Context(), email)
	if isNoRows(err) {
		ok()
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if err := s.q.DeleteUserEmailTokens(r.Context(), store.DeleteUserEmailTokensParams{UserID: u.ID, Kind: "reset"}); err != nil {
		s.internalError(w, r, err)
		return
	}
	token := newToken()
	err = s.q.CreateEmailToken(r.Context(), store.CreateEmailTokenParams{
		TokenHash: hashToken(token), UserID: u.ID, Kind: "reset", ExpiresAt: time.Now().Add(resetTokenTTL),
	})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	link := s.cfg.FrontendURL + "/reset-password?token=" + url.QueryEscape(token)
	s.sendEmail(r.Context(), u.Email, "【BandMatch】パスワードの再設定",
		fmt.Sprintf("%s さん\n\nパスワード再設定のリクエストを受け付けました。\n下のリンクから新しいパスワードを設定してください(1時間有効)。\n\n%s", u.DisplayName, link)+mailFooter)
	ok()
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if msg := validPassword(in.Password); msg != "" {
		invalid(w, msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.q.WithTx(tx)
	userID, err := q.ConsumeEmailToken(r.Context(), store.ConsumeEmailTokenParams{TokenHash: hashToken(in.Token), Kind: "reset"})
	if isNoRows(err) {
		writeError(w, 400, "invalid_token", "このリンクは無効か、有効期限が切れています。もう一度再設定メールを送ってください。")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	// メールのリンクを開けた=アドレスの持ち主なので確認済みにし、他の端末はログアウトさせる
	err = errors.Join(
		q.SetPassword(r.Context(), store.SetPasswordParams{ID: userID, PasswordHash: ptr(string(hash))}),
		q.MarkEmailVerified(r.Context(), userID),
		q.DeleteUserSessions(r.Context(), userID),
	)
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	s.clearCookie(w, sessionCookie)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	email, ok := normalizeEmail(in.Email)
	if !ok {
		invalid(w, "メールアドレスを正しく入力してください。")
		return
	}
	u, err := s.q.GetUserByEmail(r.Context(), email)
	if isNoRows(err) {
		name := cleanText(in.DisplayName)
		if name == "" {
			name = strings.Split(email, "@")[0]
		}
		now := time.Now()
		birth := time.Date(1995, 4, 1, 0, 0, 0, 0, time.UTC)
		u, err = s.q.CreateUser(r.Context(), store.CreateUserParams{
			Email: email, DisplayName: name, EmailVerifiedAt: &now, BirthDate: &birth, Gender: ptr("no_answer"),
		})
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if err := s.startSession(r.Context(), w, u.ID); err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"user": meJSON(&u)})
}

// ---- Google ログイン ----

func (s *Server) loginRedirect(w http.ResponseWriter, r *http.Request, errCode string) {
	http.Redirect(w, r, s.cfg.FrontendURL+"/login?error="+errCode, http.StatusFound)
}

func (s *Server) handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if s.oauth == nil {
		s.loginRedirect(w, r, "google_unavailable")
		return
	}
	state, verifier := newToken(), oauth2.GenerateVerifier()
	s.setCookie(w, stateCookie, state, 10*time.Minute)
	s.setCookie(w, verifierCookie, verifier, 10*time.Minute)
	http.Redirect(w, r, s.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

type googleProfile struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func (s *Server) fetchGoogleProfile(ctx context.Context, code, verifier string) (googleProfile, error) {
	var p googleProfile
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tok, err := s.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return p, err
	}
	res, err := s.oauth.Client(ctx, tok).Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		return p, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return p, fmt.Errorf("userinfo: %s", res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(&p); err != nil {
		return p, err
	}
	if p.Sub == "" || p.Email == "" {
		return p, errors.New("userinfo: sub または email がありません")
	}
	return p, nil
}

func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if s.oauth == nil {
		s.loginRedirect(w, r, "google_unavailable")
		return
	}
	stateC, err1 := r.Cookie(stateCookie)
	verC, err2 := r.Cookie(verifierCookie)
	s.clearCookie(w, stateCookie)
	s.clearCookie(w, verifierCookie)
	state := r.URL.Query().Get("state")
	if err1 != nil || err2 != nil || state == "" ||
		subtle.ConstantTimeCompare([]byte(stateC.Value), []byte(state)) != 1 || r.URL.Query().Get("code") == "" {
		s.loginRedirect(w, r, "google_failed")
		return
	}
	p, err := s.fetchGoogleProfile(r.Context(), r.URL.Query().Get("code"), verC.Value)
	if err != nil {
		slog.Warn("Google ログインに失敗", "err", err)
		s.loginRedirect(w, r, "google_failed")
		return
	}
	if !p.EmailVerified {
		s.loginRedirect(w, r, "google_email_unverified")
		return
	}
	u, err := s.upsertGoogleUser(r.Context(), p)
	if err != nil {
		slog.Error("Google ユーザーの登録に失敗", "err", err)
		s.loginRedirect(w, r, "google_failed")
		return
	}
	if err := s.startSession(r.Context(), w, u.ID); err != nil {
		slog.Error("セッション作成に失敗", "err", err)
		s.loginRedirect(w, r, "google_failed")
		return
	}
	dest := "/"
	if !profileComplete(&u) {
		dest = "/onboarding"
	}
	http.Redirect(w, r, s.cfg.FrontendURL+dest, http.StatusFound)
}

func (s *Server) upsertGoogleUser(ctx context.Context, p googleProfile) (store.User, error) {
	u, err := s.q.GetUserByGoogleSub(ctx, &p.Sub)
	if err == nil || !isNoRows(err) {
		return u, err
	}
	email := strings.ToLower(strings.TrimSpace(p.Email))
	existing, err := s.q.GetUserByEmail(ctx, email)
	if isNoRows(err) {
		name := cleanText(p.Name)
		if name == "" {
			name = strings.Split(email, "@")[0]
		}
		if r := []rune(name); len(r) > 30 {
			name = string(r[:30])
		}
		now := time.Now()
		return s.q.CreateUser(ctx, store.CreateUserParams{
			Email: email, GoogleSub: &p.Sub, DisplayName: name, AvatarUrl: p.Picture, EmailVerifiedAt: &now,
		})
	}
	if err != nil {
		return u, err
	}

	// 同じメールアドレスの既存アカウントに Google を紐づける
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return u, err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	if existing.EmailVerifiedAt == nil {
		// メール未確認のアカウントは、アドレスの持ち主ではない第三者が先に作った可能性がある。
		// その人がログインし続けられないよう、パスワードとセッションを破棄する。
		if err := errors.Join(
			q.ClearPassword(ctx, existing.ID),
			q.DeleteUserSessions(ctx, existing.ID),
			q.DeleteAllUserEmailTokens(ctx, existing.ID),
		); err != nil {
			return u, err
		}
	}
	u, err = q.LinkGoogle(ctx, store.LinkGoogleParams{ID: existing.ID, GoogleSub: &p.Sub, AvatarUrl: p.Picture})
	if err != nil {
		return u, err
	}
	return u, tx.Commit(ctx)
}

func ptr[T any](v T) *T { return &v }

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
