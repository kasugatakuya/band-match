package httpapi

import (
	"net/http"
	"time"

	"bandmatch/internal/store"
)

func dateStr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	return ptr(t.Format("2006-01-02"))
}

// meJSON は本人にだけ見せる情報を含む。
func meJSON(u *store.User) map[string]any {
	return map[string]any{
		"id": u.ID, "email": u.Email,
		"email_verified":      u.EmailVerifiedAt != nil,
		"has_password":        u.PasswordHash != nil,
		"google_linked":       u.GoogleSub != nil,
		"display_name":        u.DisplayName,
		"avatar_url":          u.AvatarUrl,
		"bio":                 u.Bio,
		"prefecture":          u.Prefecture,
		"parts":               u.Parts,
		"birth_date":          dateStr(u.BirthDate),
		"age":                 agePtr(u.BirthDate),
		"gender":              u.Gender,
		"email_notifications": u.EmailNotifications,
		"profile_complete":    profileComplete(u),
		"created_at":          u.CreatedAt,
	}
}

// publicUserJSON は誰でも見られる情報だけ(メールアドレスや生年月日は含めない)。
func publicUserJSON(u *store.User) map[string]any {
	return map[string]any{
		"id": u.ID, "display_name": u.DisplayName, "avatar_url": u.AvatarUrl, "bio": u.Bio,
		"prefecture": u.Prefecture, "parts": u.Parts, "age": agePtr(u.BirthDate), "gender": u.Gender,
		"created_at": u.CreatedAt,
	}
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var in struct {
		DisplayName        string   `json:"display_name"`
		Bio                string   `json:"bio"`
		Prefecture         string   `json:"prefecture"`
		Parts              []string `json:"parts"`
		BirthDate          string   `json:"birth_date"`
		Gender             string   `json:"gender"`
		EmailNotifications *bool    `json:"email_notifications"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	name, bio := cleanText(in.DisplayName), cleanText(in.Bio)
	if name == "" || runeLen(name) > 30 {
		invalid(w, "表示名は1〜30文字で入力してください。")
		return
	}
	if runeLen(bio) > 1000 {
		invalid(w, "自己紹介は1000文字以内で入力してください。")
		return
	}
	if in.Prefecture != "" && !contains(prefectures, in.Prefecture) {
		invalid(w, "地域を選択してください。")
		return
	}
	parts, ok := cleanChoices(in.Parts, partOptions, len(partOptions))
	if !ok {
		invalid(w, "パートの指定が正しくありません。")
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
	notify := u.EmailNotifications
	if in.EmailNotifications != nil {
		notify = *in.EmailNotifications
	}
	updated, err := s.q.UpdateProfile(r.Context(), store.UpdateProfileParams{
		ID: u.ID, DisplayName: name, Bio: bio, Prefecture: in.Prefecture, Parts: parts,
		EmailNotifications: notify, BirthDate: &birth, Gender: &in.Gender,
	})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"user": meJSON(&updated)})
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	u, err := s.q.GetUserByID(r.Context(), id)
	if isNoRows(err) {
		writeError(w, 404, "not_found", "ユーザーが見つかりませんでした。")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	rows, err := s.q.ListPostsByUser(r.Context(), store.ListPostsByUserParams{UserID: id, IncludeClosed: false})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	posts := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		posts = append(posts, postJSON(row.Post, row.AuthorName, row.AuthorAvatar, row.AuthorBirthDate, row.AuthorGender))
	}
	writeJSON(w, 200, map[string]any{"user": publicUserJSON(&u), "posts": posts})
}
