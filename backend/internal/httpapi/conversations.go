package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/google/uuid"

	"bandmatch/internal/store"
)

const (
	maxMessageLen          = 2000
	maxMessagesPerMinute   = 20
	maxNewThreadsPerDay    = 30
	notificationSnippetLen = 200
)

func messageJSON(m store.Message) map[string]any {
	return map[string]any{"id": m.ID, "sender_id": m.SenderID, "body": m.Body, "created_at": m.CreatedAt}
}

func readMessageBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	var in struct {
		Body string `json:"body"`
	}
	if !readJSON(w, r, &in) {
		return "", false
	}
	body := cleanText(in.Body)
	if body == "" || runeLen(body) > maxMessageLen {
		invalid(w, fmt.Sprintf("メッセージは1〜%d文字で入力してください。", maxMessageLen))
		return "", false
	}
	return body, true
}

func (s *Server) checkMessageRate(w http.ResponseWriter, r *http.Request, userID uuid.UUID) bool {
	n, err := s.q.CountRecentMessagesBySender(r.Context(), userID)
	if err != nil {
		s.internalError(w, r, err)
		return false
	}
	if n >= maxMessagesPerMinute {
		tooMany(w)
		return false
	}
	return true
}

// postMessage はメッセージを保存し、送信者側を既読にする。
func postMessage(ctx context.Context, q *store.Queries, convID, senderID uuid.UUID, body string) (store.Message, error) {
	m, err := q.CreateMessage(ctx, store.CreateMessageParams{ConversationID: convID, SenderID: senderID, Body: body})
	if err != nil {
		return m, err
	}
	if err := q.TouchConversation(ctx, convID); err != nil {
		return m, err
	}
	return m, q.MarkConversationRead(ctx, store.MarkConversationReadParams{ConversationID: convID, UserID: senderID})
}

// notifyRecipient は相手にメールで知らせる。
// 相手がそのスレッドを開くまでは最初の1通だけ送る(連投のたびに送らない)。
func (s *Server) notifyRecipient(ctx context.Context, convID uuid.UUID, sender *store.User, body string) {
	conv, err := s.q.GetConversationForUser(ctx, store.GetConversationForUserParams{ConversationID: convID, UserID: sender.ID})
	if err != nil || !conv.OtherEmailNotifications {
		return
	}
	n, err := s.q.ClaimNotification(ctx, store.ClaimNotificationParams{ConversationID: convID, UserID: conv.OtherID})
	if err != nil || n == 0 {
		return
	}
	snippet := body
	if r := []rune(snippet); len(r) > notificationSnippetLen {
		snippet = string(r[:notificationSnippetLen]) + "…"
	}
	about := ""
	if conv.PostTitle != "" {
		about = fmt.Sprintf("記事「%s」について、", conv.PostTitle)
	}
	text := fmt.Sprintf("%s さん\n\n%s%s さんからメッセージが届きました。\n\n----\n%s\n----\n\n返信はこちらから:\n%s/messages/%s\n\nメール通知はマイページでオフにできます。",
		conv.OtherName, about, sender.DisplayName, snippet, s.cfg.FrontendURL, convID)
	s.sendEmail(ctx, conv.OtherEmail, fmt.Sprintf("【BandMatch】%s さんからメッセージが届きました", sender.DisplayName), text+mailFooter)
}

func (s *Server) handleStartConversation(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	postID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	body, ok := readMessageBody(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	post, err := s.q.GetPost(ctx, postID)
	if isNoRows(err) {
		writeError(w, 404, "not_found", "記事が見つかりませんでした。")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if post.Post.UserID == u.ID {
		invalid(w, "自分の記事にはメッセージを送れません。")
		return
	}
	if !s.checkMessageRate(w, r, u.ID) {
		return
	}

	lookup := store.GetConversationByPostAndStarterParams{PostID: &postID, StarterID: u.ID}
	conv, err := s.q.GetConversationByPostAndStarter(ctx, lookup)
	if err != nil && !isNoRows(err) {
		s.internalError(w, r, err)
		return
	}
	isNew := isNoRows(err)
	if isNew {
		if post.Post.Status != "open" {
			writeError(w, 409, "post_closed", "この記事は募集を締め切っています。")
			return
		}
		n, err := s.q.CountRecentConversationsByStarter(ctx, u.ID)
		if err != nil {
			s.internalError(w, r, err)
			return
		}
		if n >= maxNewThreadsPerDay {
			writeError(w, 429, "rate_limited", "1日に新しく連絡できる件数の上限に達しました。")
			return
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	if isNew {
		conv, err = q.CreateConversation(ctx, store.CreateConversationParams{PostID: &postID, StarterID: u.ID, OwnerID: post.Post.UserID})
		if isUniqueViolation(err) {
			// ほぼ同時に2回押された場合。先に作られた方を使ってもらう
			writeError(w, 409, "conflict", "送信が重複しました。メッセージ一覧を確認してください。")
			return
		}
		for _, member := range []uuid.UUID{u.ID, post.Post.UserID} {
			if err == nil {
				err = q.AddConversationMember(ctx, store.AddConversationMemberParams{ConversationID: conv.ID, UserID: member})
			}
		}
		if err != nil {
			s.internalError(w, r, err)
			return
		}
	}
	if _, err := postMessage(ctx, q, conv.ID, u.ID, body); err != nil {
		s.internalError(w, r, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.internalError(w, r, err)
		return
	}
	s.notifyRecipient(ctx, conv.ID, u, body)
	writeJSON(w, 201, map[string]any{"conversation_id": conv.ID})
}

func conversationJSON(id uuid.UUID, postID *uuid.UUID, postTitle, postKind string, otherID uuid.UUID, otherName, otherAvatar string) map[string]any {
	return map[string]any{
		"id": id, "post_id": postID, "post_title": postTitle, "post_kind": postKind,
		"other": map[string]any{"id": otherID, "display_name": otherName, "avatar_url": otherAvatar},
	}
}

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListConversationsForUser(r.Context(), currentUser(r).ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, c := range rows {
		j := conversationJSON(c.ID, c.PostID, c.PostTitle, c.PostKind, c.OtherID, c.OtherName, c.OtherAvatar)
		j["last_message_at"], j["last_body"], j["unread"] = c.LastMessageAt, c.LastBody, c.Unread
		out = append(out, j)
	}
	writeJSON(w, 200, map[string]any{"conversations": out})
}

func (s *Server) handleUnreadCount(w http.ResponseWriter, r *http.Request) {
	n, err := s.q.CountUnreadForUser(r.Context(), currentUser(r).ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]int64{"unread": n})
}

// memberConversation は、ログイン中のユーザーが参加している会話だけを返す。
// 参加していない会話は存在しないものとして 404 にする。
func (s *Server) memberConversation(w http.ResponseWriter, r *http.Request) (store.GetConversationForUserRow, bool) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return store.GetConversationForUserRow{}, false
	}
	conv, err := s.q.GetConversationForUser(r.Context(), store.GetConversationForUserParams{ConversationID: id, UserID: currentUser(r).ID})
	if isNoRows(err) {
		writeError(w, 404, "not_found", "メッセージが見つかりませんでした。")
		return conv, false
	}
	if err != nil {
		s.internalError(w, r, err)
		return conv, false
	}
	return conv, true
}

func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	c, ok := s.memberConversation(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, map[string]any{
		"conversation": conversationJSON(c.ID, c.PostID, c.PostTitle, c.PostKind, c.OtherID, c.OtherName, c.OtherAvatar),
	})
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	c, ok := s.memberConversation(w, r)
	if !ok {
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	rows, err := s.q.ListMessages(r.Context(), store.ListMessagesParams{ConversationID: c.ID, AfterID: after})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	slices.Reverse(rows) // 新しい順で取っているので古い順に直す
	out := make([]map[string]any, 0, len(rows))
	for _, m := range rows {
		out = append(out, messageJSON(m))
	}
	writeJSON(w, 200, map[string]any{"messages": out})
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	c, ok := s.memberConversation(w, r)
	if !ok {
		return
	}
	err := s.q.MarkConversationRead(r.Context(), store.MarkConversationReadParams{ConversationID: c.ID, UserID: currentUser(r).ID})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	c, ok := s.memberConversation(w, r)
	if !ok {
		return
	}
	body, ok := readMessageBody(w, r)
	if !ok {
		return
	}
	if !s.checkMessageRate(w, r, u.ID) {
		return
	}
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	defer tx.Rollback(ctx)
	m, err := postMessage(ctx, s.q.WithTx(tx), c.ID, u.ID, body)
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	s.notifyRecipient(ctx, c.ID, u, body)
	writeJSON(w, 201, map[string]any{"message": messageJSON(m)})
}
