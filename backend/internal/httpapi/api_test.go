package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bandmatch/internal/config"
	"bandmatch/internal/db"
)

// 結合テスト。docker compose の Postgres に bandmatch_test という DB を作って使う。
// 接続できない環境ではスキップする。

const origin = "http://app.test"

type sentMail struct{ to, subject, body string }

type fakeMailer struct {
	mu   sync.Mutex
	sent []sentMail
}

func (f *fakeMailer) Send(_ context.Context, to, subject, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sentMail{to, subject, body})
	return nil
}

func (f *fakeMailer) to(addr string) []sentMail {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []sentMail
	for _, m := range f.sent {
		if m.to == addr {
			out = append(out, m)
		}
	}
	return out
}

type env struct {
	t    *testing.T
	url  string
	mail *fakeMailer
	pool *pgxpool.Pool
}

func setup(t *testing.T, appEnv string) *env {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		base = "postgres://bandmatch:bandmatch@localhost:54329/bandmatch?sslmode=disable"
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, base)
	if err == nil {
		err = admin.Ping(ctx)
	}
	if err != nil {
		t.Skipf("テスト用 DB に接続できないためスキップ: %v", err)
	}
	_, _ = admin.Exec(ctx, "CREATE DATABASE bandmatch_test")
	admin.Close()

	u, _ := url.Parse(base)
	u.Path = "/bandmatch_test"
	pool, err := db.Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "TRUNCATE users, email_outbox RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	mail := &fakeMailer{}
	srv := httptest.NewServer(New(config.Config{Env: appEnv, FrontendURL: origin}, pool, mail).Handler())
	t.Cleanup(srv.Close)
	return &env{t: t, url: srv.URL, mail: mail, pool: pool}
}

// client は Cookie を保持する1人分のブラウザ。
type client struct {
	e *env
	c *http.Client
}

func (e *env) newClient() *client {
	jar, _ := cookiejar.New(nil)
	return &client{e: e, c: &http.Client{Jar: jar}}
}

func (c *client) do(method, path string, body any, wantStatus int) map[string]any {
	c.e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, c.e.url+"/api"+path, &buf)
	req.Header.Set("Origin", origin)
	res, err := c.c.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	if res.StatusCode != wantStatus {
		c.e.t.Fatalf("%s %s: status %d (want %d): %v", method, path, res.StatusCode, wantStatus, out)
	}
	return out
}

var tokenRe = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

func (e *env) lastToken(addr string) string {
	e.t.Helper()
	mails := e.mail.to(addr)
	if len(mails) == 0 {
		e.t.Fatalf("%s 宛のメールがありません", addr)
	}
	m := tokenRe.FindStringSubmatch(mails[len(mails)-1].body)
	if m == nil {
		e.t.Fatalf("メールにトークンがありません: %s", mails[len(mails)-1].body)
	}
	return m[1]
}

func signupBody(email string) map[string]any {
	return map[string]any{
		"email": email, "password": "password123", "display_name": strings.Split(email, "@")[0],
		"birth_date": "1995-04-01", "gender": "male",
	}
}

// signupVerified は登録してメール確認まで済ませたユーザーを作る。
func (e *env) signupVerified(email string) (*client, string) {
	e.t.Helper()
	c := e.newClient()
	res := c.do("POST", "/auth/signup", signupBody(email), 201)
	c.do("POST", "/auth/verify-email", map[string]any{"token": e.lastToken(email)}, 200)
	return c, res["user"].(map[string]any)["id"].(string)
}

func postBody(title string) map[string]any {
	return map[string]any{
		"kind": "recruit", "title": title, "body": "一緒にやりましょう", "prefecture": "東京都",
		"parts": []string{"ドラム"}, "genres": []string{"ロック"}, "artists": []string{"Number Girl"},
	}
}

func createPost(c *client, body map[string]any) string {
	return c.do("POST", "/posts", body, 201)["post"].(map[string]any)["id"].(string)
}

func TestSignupVerifyLogin(t *testing.T) {
	e := setup(t, "production")
	c := e.newClient()

	// 必須項目(年齢・性別)がないと登録できない
	bad := signupBody("a@example.com")
	delete(bad, "birth_date")
	c.do("POST", "/auth/signup", bad, 422)
	bad = signupBody("a@example.com")
	bad["gender"] = ""
	c.do("POST", "/auth/signup", bad, 422)
	bad = signupBody("a@example.com")
	bad["password"] = "short"
	c.do("POST", "/auth/signup", bad, 422)

	res := c.do("POST", "/auth/signup", signupBody("A@Example.com"), 201)
	user := res["user"].(map[string]any)
	if user["email"] != "a@example.com" || user["email_verified"] != false || user["age"] == nil {
		t.Fatalf("登録直後のユーザーが想定と違う: %v", user)
	}
	c.do("POST", "/auth/signup", signupBody("a@example.com"), 409)

	// メール確認前は投稿できない
	if got := c.do("POST", "/posts", postBody("募集"), 403)["code"]; got != "email_unverified" {
		t.Fatalf("code = %v", got)
	}
	token := e.lastToken("a@example.com")
	c.do("POST", "/auth/verify-email", map[string]any{"token": token}, 200)
	c.do("POST", "/auth/verify-email", map[string]any{"token": token}, 400) // 使い回し不可
	createPost(c, postBody("募集"))

	c.do("POST", "/auth/logout", nil, 200)
	if c.do("GET", "/me", nil, 200)["user"] != nil {
		t.Fatal("ログアウト後もログイン状態になっている")
	}
	c.do("POST", "/posts", postBody("募集"), 401)
	c.do("POST", "/auth/login", map[string]any{"email": "a@example.com", "password": "wrong-password"}, 401)
	c.do("POST", "/auth/login", map[string]any{"email": "nobody@example.com", "password": "password123"}, 401)
	c.do("POST", "/auth/login", map[string]any{"email": "a@example.com", "password": "password123"}, 200)
	if c.do("GET", "/me", nil, 200)["user"] == nil {
		t.Fatal("ログインできていない")
	}

	// 本番では開発用ログインは存在しない
	req, _ := http.NewRequest("POST", e.url+"/api/auth/dev-login", strings.NewReader(`{"email":"x@example.com"}`))
	req.Header.Set("Origin", origin)
	r, _ := http.DefaultClient.Do(req)
	if r.StatusCode != 404 && r.StatusCode != 405 {
		t.Fatalf("dev-login が本番で有効: %d", r.StatusCode)
	}
}

func TestPasswordReset(t *testing.T) {
	e := setup(t, "production")
	c, _ := e.signupVerified("a@example.com")

	e.newClient().do("POST", "/auth/password/forgot", map[string]any{"email": "nobody@example.com"}, 200)
	if len(e.mail.to("nobody@example.com")) != 0 {
		t.Fatal("未登録アドレスにメールを送っている")
	}
	e.newClient().do("POST", "/auth/password/forgot", map[string]any{"email": "a@example.com"}, 200)
	token := e.lastToken("a@example.com")
	other := e.newClient()
	other.do("POST", "/auth/password/reset", map[string]any{"token": token, "password": "new-password-1"}, 200)
	other.do("POST", "/auth/password/reset", map[string]any{"token": token, "password": "new-password-2"}, 400)

	// 再設定すると既存のログインは無効になる
	if c.do("GET", "/me", nil, 200)["user"] != nil {
		t.Fatal("再設定後も古いセッションが有効")
	}
	c.do("POST", "/auth/login", map[string]any{"email": "a@example.com", "password": "password123"}, 401)
	c.do("POST", "/auth/login", map[string]any{"email": "a@example.com", "password": "new-password-1"}, 200)
}

func TestOriginCheck(t *testing.T) {
	e := setup(t, "production")
	for _, o := range []string{"", "http://evil.test"} {
		req, _ := http.NewRequest("POST", e.url+"/api/auth/login", strings.NewReader(`{}`))
		if o != "" {
			req.Header.Set("Origin", o)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 403 {
			t.Fatalf("Origin %q: status %d (want 403)", o, res.StatusCode)
		}
	}
}

func TestPostsAndSearch(t *testing.T) {
	e := setup(t, "production")
	a, _ := e.signupVerified("a@example.com")
	b, _ := e.signupVerified("b@example.com")

	id1 := createPost(a, postBody("ドラム募集"))
	p2 := postBody("ギターで加入希望")
	p2["kind"], p2["prefecture"] = "join", "大阪府"
	p2["parts"], p2["genres"], p2["artists"] = []string{"ギター"}, []string{"ジャズ"}, []string{"Pat Metheny", "100%オレンジ"}
	id2 := createPost(b, p2)

	invalidPost := postBody("x")
	invalidPost["parts"] = []string{"存在しないパート"}
	a.do("POST", "/posts", invalidPost, 422)

	cases := []struct {
		query string
		want  []string
	}{
		{"", []string{id2, id1}},
		{"?kind=recruit", []string{id1}},
		{"?kind=join", []string{id2}},
		{"?prefecture=" + url.QueryEscape("大阪府"), []string{id2}},
		{"?part=" + url.QueryEscape("ドラム"), []string{id1}},
		{"?genre=" + url.QueryEscape("ジャズ"), []string{id2}},
		{"?artist=metheny", []string{id2}},
		{"?artist=number", []string{id1}},
		{"?artist=" + url.QueryEscape("%"), []string{id2}}, // % は文字として扱う
		{"?q=" + url.QueryEscape("加入"), []string{id2}},
		{"?kind=recruit&genre=" + url.QueryEscape("ジャズ"), nil},
	}
	anon := e.newClient()
	for _, tc := range cases {
		res := anon.do("GET", "/posts"+tc.query, nil, 200)
		var got []string
		for _, p := range res["posts"].([]any) {
			got = append(got, p.(map[string]any)["id"].(string))
		}
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("GET /posts%s = %v, want %v", tc.query, got, tc.want)
		}
	}

	// ページング
	res := anon.do("GET", "/posts?limit=1", nil, 200)
	cursor, _ := res["next_cursor"].(string)
	if cursor == "" {
		t.Fatal("next_cursor がない")
	}
	res = anon.do("GET", "/posts?limit=1&cursor="+url.QueryEscape(cursor), nil, 200)
	if got := res["posts"].([]any)[0].(map[string]any)["id"]; got != id1 || res["next_cursor"] != nil {
		t.Fatalf("2ページ目が想定と違う: %v", res)
	}

	// 公開情報にメールアドレスや生年月日が含まれない
	raw, _ := json.Marshal(anon.do("GET", "/posts/"+id1, nil, 200))
	if strings.Contains(string(raw), "example.com") || strings.Contains(string(raw), "1995") {
		t.Fatalf("公開 API に個人情報が含まれている: %s", raw)
	}

	// 他人の記事は編集も削除もできない
	upd := postBody("乗っ取り")
	upd["status"] = "open"
	b.do("PATCH", "/posts/"+id1, upd, 404)
	b.do("DELETE", "/posts/"+id1, nil, 404)
	anon.do("PATCH", "/posts/"+id1, upd, 401)

	// 締め切ると一覧から消えるが、本人の一覧には残る
	upd = postBody("ドラム募集(締切)")
	upd["status"] = "closed"
	a.do("PATCH", "/posts/"+id1, upd, 200)
	if n := len(anon.do("GET", "/posts?kind=recruit", nil, 200)["posts"].([]any)); n != 0 {
		t.Fatalf("締め切った記事が一覧に出ている")
	}
	if n := len(a.do("GET", "/me/posts", nil, 200)["posts"].([]any)); n != 1 {
		t.Fatalf("自分の記事一覧 = %d 件", n)
	}
	a.do("DELETE", "/posts/"+id1, nil, 200)
	anon.do("GET", "/posts/"+id1, nil, 404)
}

func TestMessagingAndNotifications(t *testing.T) {
	e := setup(t, "production")
	a, aID := e.signupVerified("a@example.com") // 募集した人
	b, bID := e.signupVerified("b@example.com") // 応募する人
	c, _ := e.signupVerified("c@example.com")   // 無関係な人
	postID := createPost(a, postBody("ドラム募集"))
	notifications := func(addr string) int {
		n := 0
		for _, m := range e.mail.to(addr) {
			if strings.Contains(m.subject, "メッセージが届きました") {
				n++
			}
		}
		return n
	}

	a.do("POST", "/posts/"+postID+"/conversations", map[string]any{"body": "自分宛"}, 422)
	e.newClient().do("POST", "/posts/"+postID+"/conversations", map[string]any{"body": "hi"}, 401)
	b.do("POST", "/posts/"+postID+"/conversations", map[string]any{"body": "  "}, 422)

	convID := b.do("POST", "/posts/"+postID+"/conversations", map[string]any{"body": "応募したいです"}, 201)["conversation_id"].(string)
	if notifications("a@example.com") != 1 {
		t.Fatalf("最初のメッセージで通知が届いていない")
	}
	mails := e.mail.to("a@example.com")
	if last := mails[len(mails)-1]; !strings.Contains(last.body, "/messages/"+convID) || !strings.Contains(last.body, "応募したいです") {
		t.Fatalf("通知メールの内容が想定と違う: %s", last.body)
	}

	// 同じ記事にもう一度送っても同じスレッドが使われ、未読の間は追加で通知しない
	again := b.do("POST", "/posts/"+postID+"/conversations", map[string]any{"body": "追伸です"}, 201)["conversation_id"]
	if again != convID {
		t.Fatalf("スレッドが重複して作られた")
	}
	b.do("POST", "/conversations/"+convID+"/messages", map[string]any{"body": "土曜はどうですか"}, 201)
	if n := notifications("a@example.com"); n != 1 {
		t.Fatalf("未読の間に通知が %d 通届いた(1通のはず)", n)
	}

	// 未読数
	if n := a.do("GET", "/conversations/unread-count", nil, 200)["unread"]; n != float64(3) {
		t.Fatalf("A の未読 = %v, want 3", n)
	}
	if n := b.do("GET", "/conversations/unread-count", nil, 200)["unread"]; n != float64(0) {
		t.Fatalf("B の未読 = %v, want 0", n)
	}
	list := a.do("GET", "/conversations", nil, 200)["conversations"].([]any)
	first := list[0].(map[string]any)
	if len(list) != 1 || first["unread"] != float64(3) || first["last_body"] != "土曜はどうですか" ||
		first["other"].(map[string]any)["id"] != bID || first["post_title"] != "ドラム募集" {
		t.Fatalf("会話一覧が想定と違う: %v", list)
	}

	// 参加者以外は見ることも送ることもできない
	c.do("GET", "/conversations/"+convID, nil, 404)
	c.do("GET", "/conversations/"+convID+"/messages", nil, 404)
	c.do("POST", "/conversations/"+convID+"/messages", map[string]any{"body": "割り込み"}, 404)
	c.do("POST", "/conversations/"+convID+"/read", nil, 404)
	if n := len(c.do("GET", "/conversations", nil, 200)["conversations"].([]any)); n != 0 {
		t.Fatalf("無関係な人の会話一覧に表示されている")
	}

	// A が読んで返信 → B に通知。A が既読にしたので、次の B のメッセージでまた A に通知
	msgs := a.do("GET", "/conversations/"+convID+"/messages", nil, 200)["messages"].([]any)
	if len(msgs) != 3 || msgs[0].(map[string]any)["body"] != "応募したいです" || msgs[0].(map[string]any)["sender_id"] != bID {
		t.Fatalf("メッセージ一覧が想定と違う: %v", msgs)
	}
	a.do("POST", "/conversations/"+convID+"/read", nil, 200)
	if n := a.do("GET", "/conversations/unread-count", nil, 200)["unread"]; n != float64(0) {
		t.Fatalf("既読後の未読 = %v", n)
	}
	reply := a.do("POST", "/conversations/"+convID+"/messages", map[string]any{"body": "土曜14時でお願いします"}, 201)["message"].(map[string]any)
	if reply["sender_id"] != aID {
		t.Fatalf("送信者が違う: %v", reply)
	}
	if notifications("b@example.com") != 1 {
		t.Fatal("返信の通知が B に届いていない")
	}
	lastID := int64(msgs[2].(map[string]any)["id"].(float64))
	newer := b.do("GET", "/conversations/"+convID+"/messages?after="+strings.TrimSpace(jsonInt(lastID)), nil, 200)["messages"].([]any)
	if len(newer) != 1 || newer[0].(map[string]any)["body"] != "土曜14時でお願いします" {
		t.Fatalf("after 指定の取得が想定と違う: %v", newer)
	}
	b.do("POST", "/conversations/"+convID+"/messages", map[string]any{"body": "了解です"}, 201)
	if n := notifications("a@example.com"); n != 2 {
		t.Fatalf("既読後の新着で通知が届いていない: %d 通", n)
	}

	// 通知をオフにした相手には送らない
	b.do("PATCH", "/me", map[string]any{
		"display_name": "b", "birth_date": "1995-04-01", "gender": "male", "email_notifications": false,
	}, 200)
	b.do("POST", "/conversations/"+convID+"/read", nil, 200)
	a.do("POST", "/conversations/"+convID+"/messages", map[string]any{"body": "スタジオ予約しました"}, 201)
	if n := notifications("b@example.com"); n != 1 {
		t.Fatalf("通知オフなのに届いた: %d 通", n)
	}

	// 送信記録が残っている
	var sent int
	if err := e.pool.QueryRow(context.Background(), "SELECT count(*) FROM email_outbox WHERE status = 'sent'").Scan(&sent); err != nil {
		t.Fatal(err)
	}
	if sent != len(e.mail.sent) {
		t.Fatalf("outbox の送信済み %d 件 / 実際の送信 %d 件", sent, len(e.mail.sent))
	}

	// 締め切った記事には新しく連絡できないが、既存のやり取りは続けられる
	upd := postBody("ドラム募集")
	upd["status"] = "closed"
	a.do("PATCH", "/posts/"+postID, upd, 200)
	c.do("POST", "/posts/"+postID+"/conversations", map[string]any{"body": "まだ募集してますか"}, 409)
	b.do("POST", "/conversations/"+convID+"/messages", map[string]any{"body": "当日よろしくお願いします"}, 201)

	// 記事を消しても会話は残る
	a.do("DELETE", "/posts/"+postID, nil, 200)
	if n := len(b.do("GET", "/conversations/"+convID+"/messages", nil, 200)["messages"].([]any)); n != 7 {
		t.Fatalf("記事削除後のメッセージ数 = %d", n)
	}
}

func TestProfileRequiredAndDevLogin(t *testing.T) {
	e := setup(t, "development")
	c := e.newClient()
	c.do("POST", "/auth/dev-login", map[string]any{"email": "dev@example.com", "display_name": "開発"}, 200)
	createPost(c, postBody("開発用ログインで投稿"))

	// Google ログイン直後を再現: 生年月日と性別が未入力だと投稿できない
	if _, err := e.pool.Exec(context.Background(), "UPDATE users SET birth_date = NULL, gender = NULL"); err != nil {
		t.Fatal(err)
	}
	if got := c.do("POST", "/posts", postBody("募集"), 403)["code"]; got != "profile_incomplete" {
		t.Fatalf("code = %v", got)
	}
	c.do("PATCH", "/me", map[string]any{"display_name": "開発", "gender": "female"}, 422)
	c.do("PATCH", "/me", map[string]any{"display_name": "開発", "birth_date": time.Now().Format("2006-01-02"), "gender": "female"}, 422)
	me := c.do("PATCH", "/me", map[string]any{"display_name": "開発", "birth_date": "2000-01-01", "gender": "female"}, 200)
	if me["user"].(map[string]any)["profile_complete"] != true {
		t.Fatalf("プロフィールが完了扱いになっていない: %v", me)
	}
	createPost(c, postBody("募集"))
}

func jsonInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestPostLimitPerKind(t *testing.T) {
	e := setup(t, "production")
	a, _ := e.signupVerified("a@example.com")
	b, _ := e.signupVerified("b@example.com")

	join := func(title string) map[string]any {
		p := postBody(title)
		p["kind"] = "join"
		return p
	}

	// メンバー募集は3件まで
	var recruitIDs []string
	for i := 0; i < 3; i++ {
		recruitIDs = append(recruitIDs, createPost(a, postBody(fmt.Sprintf("募集%d", i))))
	}
	res := a.do("POST", "/posts", postBody("募集4件目"), 409)
	if res["code"] != "post_limit" {
		t.Fatalf("code = %v, want post_limit", res["code"])
	}

	// 加入希望は別枠で3件まで
	var joinIDs []string
	for i := 0; i < 3; i++ {
		joinIDs = append(joinIDs, createPost(a, join(fmt.Sprintf("加入%d", i))))
	}
	a.do("POST", "/posts", join("加入4件目"), 409)

	// 上限は人ごと
	createPost(b, postBody("別の人の募集"))

	// 上限に達していても、既存の記事の編集と締め切りはできる
	edit := postBody("募集0を編集")
	edit["status"] = "closed"
	a.do("PATCH", "/posts/"+recruitIDs[0], edit, 200)

	// 締め切った記事も数に含まれる
	a.do("POST", "/posts", postBody("締め切り後の募集"), 409)

	// 種類を変えて上限を超えることはできない
	toJoin := join("募集を加入希望に変更")
	toJoin["status"] = "open"
	a.do("PATCH", "/posts/"+recruitIDs[1], toJoin, 409)

	// 削除すると枠が空く
	a.do("DELETE", "/posts/"+joinIDs[0], nil, 200)
	a.do("PATCH", "/posts/"+recruitIDs[1], toJoin, 200)
	createPost(a, postBody("枠が空いたので募集"))
}
