// Package mailer はメール送信の実装(SMTP / Resend / ログ出力)をまとめる。
package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// Log はメールを送らずログに出すだけの実装。
type Log struct{}

func (Log) Send(_ context.Context, to, subject, body string) error {
	slog.Info("メール(ログ出力のみ)", "to", to, "subject", subject, "body", body)
	return nil
}

// SMTP は認証なしの SMTP サーバー(ローカルの Mailpit)向け。
type SMTP struct {
	Addr string
	From string
}

func (m SMTP) Send(ctx context.Context, to, subject, body string) error {
	from, err := mail.ParseAddress(m.From)
	if err != nil {
		return fmt.Errorf("MAIL_FROM が不正: %w", err)
	}
	if strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("宛先が不正です")
	}
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", from.String())
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.BEncoding.Encode("UTF-8", subject))
	fmt.Fprintf(&msg, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	msg.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n")
	msg.WriteString(strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n"))

	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", m.Addr)
	if err != nil {
		return err
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	host, _, _ := net.SplitHostPort(m.Addr)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if err := c.Mail(from.Address); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg.Bytes()); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// Resend は Resend の HTTP API で送信する本番用の実装。
type Resend struct {
	APIKey string
	From   string
	Client *http.Client
}

func (m Resend) Send(ctx context.Context, to, subject, body string) error {
	payload, _ := json.Marshal(map[string]any{
		"from": m.From, "to": []string{to}, "subject": subject, "text": body,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := m.Client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("resend: %s: %s", res.Status, b)
	}
	return nil
}
