package httpapi

import (
	"context"
	"log/slog"
	"time"

	"bandmatch/internal/store"
)

// sendEmail は送信内容を email_outbox に記録してからその場で送る。
// 失敗したものは RunBackground が後で再送する。
func (s *Server) sendEmail(ctx context.Context, to, subject, body string) {
	id, err := s.q.EnqueueEmail(ctx, store.EnqueueEmailParams{ToEmail: to, Subject: subject, Body: body})
	if err != nil {
		slog.Error("メールの登録に失敗", "err", err)
		return
	}
	s.deliver(ctx, id, to, subject, body)
}

func (s *Server) deliver(ctx context.Context, id int64, to, subject, body string) {
	// リクエストが先に終わっても送信と結果の記録は最後まで行う
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := s.mail.Send(ctx, to, subject, body); err != nil {
		slog.Warn("メール送信に失敗(後で再送)", "id", id, "err", err)
		if err := s.q.MarkEmailFailed(ctx, id); err != nil {
			slog.Error("メール送信失敗の記録に失敗", "id", id, "err", err)
		}
		return
	}
	if err := s.q.MarkEmailSent(ctx, id); err != nil {
		slog.Error("メール送信済みの記録に失敗", "id", id, "err", err)
	}
}

func (s *Server) retryEmails(ctx context.Context) {
	rows, err := s.q.ListRetryableEmails(ctx)
	if err != nil {
		if ctx.Err() == nil {
			slog.Warn("再送対象メールの取得に失敗", "err", err)
		}
		return
	}
	for _, e := range rows {
		s.deliver(ctx, e.ID, e.ToEmail, e.Subject, e.Body)
	}
}

const mailFooter = "\n\n--\nBandMatch\nこのメールに心当たりがない場合は破棄してください。"
