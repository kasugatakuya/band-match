# BandMatch

バンドメンバーを探している人と、バンドに入りたい人をつなぐ募集サイト。

- アカウント登録・ログイン(メールアドレス+パスワード / Google)。生年月日(公開は年齢のみ)と性別は必須
- 「メンバー募集」「加入希望」の記事を投稿し、地域・楽器パート・ジャンル・好きなアーティスト・キーワードで検索
- 記事から投稿者にメッセージを送り、1対1のチャットで練習日程などを相談
- メッセージが届くとメールで通知(相手がスレッドを開くまでは最初の1通だけ)

## 構成

| | 技術 | 本番の想定 |
|---|---|---|
| `frontend/` | Next.js (App Router) + TypeScript + Tailwind CSS | Vercel |
| `backend/` | Go + chi + pgx + sqlc + goose | Cloud Run |
| DB | PostgreSQL | Neon |
| メール | ローカルは Mailpit、本番は Resend | Resend |

ブラウザは API を必ずフロントのドメインの `/api/*` 経由で呼びます(`frontend/next.config.ts` の rewrites が Go に転送)。
ログイン Cookie を自サイトのものとして扱うためで、CORS の設定は不要です。

## ローカルで動かす

必要なもの: Docker、Go、Node.js、pnpm

```sh
docker compose up -d                 # PostgreSQL(54329) と Mailpit(8025) を起動

cd backend
cp .env.example .env                 # 初回のみ
go run ./cmd/server                  # http://localhost:8080 (起動時に DB マイグレーションを自動適用)

cd frontend
pnpm install
pnpm dev                             # http://localhost:3000
```

- 送信されたメール(確認メール・通知メール)は http://localhost:8025 で見られます。実際には送信されません
- ログイン画面の「開発用ログイン」で、登録なしにテストユーザー2人を切り替えられます(`APP_ENV=development` のときだけ有効)

### Google ログインを有効にする

1. [Google Cloud コンソール](https://console.cloud.google.com/) でプロジェクトを作り、「API とサービス > OAuth 同意画面」を設定
2. 「認証情報 > 認証情報を作成 > OAuth クライアント ID」で種類「ウェブ アプリケーション」を選ぶ
3. 「承認済みのリダイレクト URI」に `http://localhost:3000/api/auth/google/callback` を追加(本番は `https://<本番ドメイン>/api/auth/google/callback` も)
4. 発行されたクライアント ID とシークレットを `backend/.env` の `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` に書いて再起動

未設定の間は Google ログインのボタン自体が表示されません。

## テスト

```sh
cd backend && make test              # go vet + 結合テスト(docker compose の PostgreSQL に bandmatch_test を作って使う)
cd frontend && pnpm lint && pnpm exec tsc --noEmit && pnpm build
```

## 開発メモ

- SQL を変えたら `cd backend && make sqlc` で `internal/store/` を再生成(生成物はコミットする)
- テーブル変更は `backend/internal/db/migrations/` に連番の SQL を追加(適用済みのファイルは書き換えない)
- 都道府県・パート・ジャンルの選択肢は `backend/internal/httpapi/meta.go` と `frontend/src/lib/constants.ts` の両方にあるので、変えるときは両方直す

本番へのデプロイ手順は [docs/DEPLOY.md](docs/DEPLOY.md) を参照。
