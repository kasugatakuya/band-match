package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"bandmatch/internal/store"
)

func postJSON(p store.Post, authorName, authorAvatar string, authorBirth *time.Time, authorGender *string) map[string]any {
	return map[string]any{
		"id": p.ID, "kind": p.Kind, "title": p.Title, "body": p.Body, "prefecture": p.Prefecture,
		"parts": p.Parts, "genres": p.Genres, "artists": p.Artists, "status": p.Status,
		"created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		"author": map[string]any{
			"id": p.UserID, "display_name": authorName, "avatar_url": authorAvatar,
			"age": agePtr(authorBirth), "gender": authorGender,
		},
	}
}

type postInput struct {
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Prefecture string   `json:"prefecture"`
	Parts      []string `json:"parts"`
	Genres     []string `json:"genres"`
	Artists    []string `json:"artists"`
	Status     string   `json:"status"`
}

// validate は入力を整えて検証する。問題があればメッセージを返す。
func (in *postInput) validate() string {
	if in.Kind != "recruit" && in.Kind != "join" {
		return "記事の種類を選択してください。"
	}
	in.Title, in.Body = cleanText(in.Title), cleanText(in.Body)
	if in.Title == "" || runeLen(in.Title) > 80 {
		return "タイトルは1〜80文字で入力してください。"
	}
	if in.Body == "" || runeLen(in.Body) > 4000 {
		return "本文は1〜4000文字で入力してください。"
	}
	if !contains(prefectures, in.Prefecture) {
		return "地域を選択してください。"
	}
	var ok bool
	if in.Parts, ok = cleanChoices(in.Parts, partOptions, len(partOptions)); !ok {
		return "パートの指定が正しくありません。"
	}
	if len(in.Parts) == 0 {
		return "パートを1つ以上選択してください。"
	}
	if in.Genres, ok = cleanChoices(in.Genres, genreOptions, len(genreOptions)); !ok {
		return "ジャンルの指定が正しくありません。"
	}
	if in.Artists, ok = cleanTags(in.Artists, 20, 50); !ok {
		return "好きなアーティストは20件まで、1件50文字以内で入力してください。"
	}
	return ""
}

func optionalQuery(r *http.Request, key string) *string {
	v := cleanText(r.URL.Query().Get(key))
	if v == "" {
		return nil
	}
	return &v
}

func optionalLike(r *http.Request, key string) *string {
	v := optionalQuery(r, key)
	if v == nil {
		return nil
	}
	if r := []rune(*v); len(r) > 100 {
		*v = string(r[:100])
	}
	return ptr(likeEscape(*v))
}

func (s *Server) handleListPosts(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n >= 1 && n <= 50 {
		limit = n
	}
	params := store.ListPostsParams{
		Kind:       optionalQuery(r, "kind"),
		Prefecture: optionalQuery(r, "prefecture"),
		Part:       optionalQuery(r, "part"),
		Genre:      optionalQuery(r, "genre"),
		Artist:     optionalLike(r, "artist"),
		Q:          optionalLike(r, "q"),
		Lim:        int32(limit + 1), // 次のページがあるか知るために1件多く取る
	}
	if c := r.URL.Query().Get("cursor"); c != "" {
		t, err := time.Parse(time.RFC3339Nano, c)
		if err != nil {
			writeError(w, 400, "bad_request", "cursor が不正です。")
			return
		}
		params.Before = &t
	}
	rows, err := s.q.ListPosts(r.Context(), params)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		next = ptr(rows[limit-1].Post.CreatedAt.Format(time.RFC3339Nano))
	}
	posts := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		posts = append(posts, postJSON(row.Post, row.AuthorName, row.AuthorAvatar, row.AuthorBirthDate, row.AuthorGender))
	}
	writeJSON(w, 200, map[string]any{"posts": posts, "next_cursor": next})
}

func (s *Server) handleGetPost(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	row, err := s.q.GetPost(r.Context(), id)
	if isNoRows(err) {
		writeError(w, 404, "not_found", "記事が見つかりませんでした。")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"post": postJSON(row.Post, row.AuthorName, row.AuthorAvatar, row.AuthorBirthDate, row.AuthorGender)})
}

func (s *Server) handleMyPosts(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	rows, err := s.q.ListPostsByUser(r.Context(), store.ListPostsByUserParams{UserID: u.ID, IncludeClosed: true})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	posts := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		posts = append(posts, postJSON(row.Post, row.AuthorName, row.AuthorAvatar, row.AuthorBirthDate, row.AuthorGender))
	}
	writeJSON(w, 200, map[string]any{"posts": posts})
}

// maxPostsPerKind は、1人が持てる記事数の上限(メンバー募集・加入希望それぞれ)。
// 締め切った記事も数える。枠を空けるには記事を削除する。
const maxPostsPerKind = 3

// checkPostLimit は、kind の記事が上限に達していたらエラーを返して false を返す。
// excludeID には編集中の記事の ID を渡す(新規作成なら uuid.Nil)。
func (s *Server) checkPostLimit(w http.ResponseWriter, r *http.Request, userID uuid.UUID, kind string, excludeID uuid.UUID) bool {
	n, err := s.q.CountPostsByUserAndKind(r.Context(), store.CountPostsByUserAndKindParams{
		UserID: userID, Kind: kind, ExcludeID: excludeID,
	})
	if err != nil {
		s.internalError(w, r, err)
		return false
	}
	if n >= maxPostsPerKind {
		label := "メンバー募集"
		if kind == "join" {
			label = "加入希望"
		}
		writeError(w, 409, "post_limit", fmt.Sprintf("%sの記事は%d件までです。新しく投稿するには、不要な記事を削除してください。", label, maxPostsPerKind))
		return false
	}
	return true
}

func (s *Server) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var in postInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		invalid(w, msg)
		return
	}
	n, err := s.q.CountRecentPostsByUser(r.Context(), u.ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if n >= 10 {
		writeError(w, 429, "rate_limited", "1日に投稿できる記事は10件までです。")
		return
	}
	if !s.checkPostLimit(w, r, u.ID, in.Kind, uuid.Nil) {
		return
	}
	p, err := s.q.CreatePost(r.Context(), store.CreatePostParams{
		UserID: u.ID, Kind: in.Kind, Title: in.Title, Body: in.Body, Prefecture: in.Prefecture,
		Parts: in.Parts, Genres: in.Genres, Artists: in.Artists,
	})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, 201, map[string]any{"post": postJSON(p, u.DisplayName, u.AvatarUrl, u.BirthDate, u.Gender)})
}

func (s *Server) handleUpdatePost(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var in postInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		invalid(w, msg)
		return
	}
	if in.Status != "open" && in.Status != "closed" {
		invalid(w, "募集状況の指定が正しくありません。")
		return
	}
	// 種類を変更して上限を超えるのを防ぐ(編集中の記事自身は数えない)
	if !s.checkPostLimit(w, r, u.ID, in.Kind, id) {
		return
	}
	// WHERE に user_id を含めているので、他人の記事は更新されず「見つからない」扱いになる
	p, err := s.q.UpdatePost(r.Context(), store.UpdatePostParams{
		ID: id, UserID: u.ID, Kind: in.Kind, Title: in.Title, Body: in.Body, Prefecture: in.Prefecture,
		Parts: in.Parts, Genres: in.Genres, Artists: in.Artists, Status: in.Status,
	})
	if isNoRows(err) {
		writeError(w, 404, "not_found", "記事が見つからないか、編集する権限がありません。")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"post": postJSON(p, u.DisplayName, u.AvatarUrl, u.BirthDate, u.Gender)})
}

func (s *Server) handleDeletePost(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	n, err := s.q.DeletePost(r.Context(), store.DeletePostParams{ID: id, UserID: u.ID})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if n == 0 {
		writeError(w, 404, "not_found", "記事が見つからないか、削除する権限がありません。")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
