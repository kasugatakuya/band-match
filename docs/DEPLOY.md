# 本番デプロイ手順

構成: Vercel(フロント)→ `/api/*` を転送 → Cloud Run(Go)→ Neon(PostgreSQL)、メールは Resend。
いずれも無料枠で始められ、アクセスがないときはほぼ課金されません。

> この手順は構成に合わせて書いたもので、実際の本番デプロイはまだ行っていません。
> 各サービスの画面や料金は変わることがあるので、作業時に公式の案内も確認してください。

## 事前に必要なもの

- **独自ドメイン**: Resend は、認証済みの自分のドメインからでないと自分以外の宛先にメールを送れません。メール通知を使うには必須です
- Google Cloud(課金を有効にしたプロジェクト)、Neon、Resend、Vercel のアカウント
- このリポジトリを GitHub に push しておく(Vercel の連携に使う)

## 1. Neon(データベース)

1. プロジェクトを作成(リージョンは日本に近いところ)
2. 接続文字列をコピーする。**プーリングなし(Direct)** の方を使い、末尾が `?sslmode=require` になっていることを確認
3. テーブルは API サーバーの起動時に自動で作られるので、手作業は不要

## 2. Resend(メール)

1. ドメインを追加し、表示された DNS レコード(SPF / DKIM)をドメインの DNS に登録して認証を完了させる
2. API キーを作成
3. 送信元は認証したドメインのアドレスにする(例: `BandMatch <no-reply@example.com>`)

## 3. Cloud Run(Go の API)

```sh
gcloud config set project <プロジェクトID>
gcloud services enable run.googleapis.com cloudbuild.googleapis.com secretmanager.googleapis.com

# 秘密の値は Secret Manager に入れる
printf '%s' '<Neon の接続文字列>'        | gcloud secrets create bandmatch-database-url --data-file=-
printf '%s' '<Resend の API キー>'       | gcloud secrets create bandmatch-resend-key --data-file=-
printf '%s' '<Google のクライアントシークレット>' | gcloud secrets create bandmatch-google-secret --data-file=-

gcloud run deploy bandmatch-api \
  --source backend \
  --region asia-northeast1 \
  --allow-unauthenticated \
  --min-instances 0 --max-instances 2 \
  --set-env-vars 'APP_ENV=production,FRONTEND_URL=https://<本番ドメイン>,MAIL_DRIVER=resend,MAIL_FROM=BandMatch <no-reply@<ドメイン>>,GOOGLE_CLIENT_ID=<クライアントID>' \
  --set-secrets 'DATABASE_URL=bandmatch-database-url:latest,RESEND_API_KEY=bandmatch-resend-key:latest,GOOGLE_CLIENT_SECRET=bandmatch-google-secret:latest'
```

- 初回は、Cloud Run のサービスアカウントに「Secret Manager のシークレット アクセサー」ロールを付けるよう求められることがあります
- `FRONTEND_URL` は**ブラウザで開くサイトの URL**(Vercel 側のドメイン)です。Cookie の Secure 属性、CSRF 対策の Origin 判定、メール内のリンク、Google ログインの戻り先に使われるので、末尾のスラッシュなしで正確に入れてください
- デプロイ後に表示される URL(`https://bandmatch-api-xxxx.a.run.app`)を控え、`/healthz` が `{"ok":true}` を返すことを確認
- Google ログインを使わない場合は `GOOGLE_CLIENT_ID` と `GOOGLE_CLIENT_SECRET` を省く

## 4. Vercel(フロント)

1. GitHub のリポジトリをインポートし、**Root Directory を `frontend`** にする
2. 環境変数 `BACKEND_URL` に Cloud Run の URL を設定
3. 独自ドメインを割り当てる(3. の `FRONTEND_URL` と同じにする)

## 5. Google ログイン

OAuth クライアントの「承認済みのリダイレクト URI」に `https://<本番ドメイン>/api/auth/google/callback` を追加し、OAuth 同意画面を「本番環境」に公開する。

## 6. 公開後の確認

- 新規登録 → 確認メールが届く → リンクで確認完了
- 記事を投稿 → 別アカウントからメッセージ → 通知メールが届く
- Google ログイン → 初回は生年月日・性別の入力画面に進む
- `https://<本番ドメイン>/api/auth/dev-login` に POST しても 404 になる(開発用ログインは本番では無効)

## 知っておくこと

- **メールの再送**: 送信に失敗したメールは `email_outbox` テーブルに残り、API サーバーが1分おきに再送します。ただし Cloud Run はリクエストがない間 CPU が止まるため、再送されるのは次にアクセスがあったときです
- **ログイン試行の回数制限**はサーバーのメモリ上で数えています。インスタンスが複数になると制限がゆるくなるので、`--max-instances` は小さくしておくのが無難です
- **公開前に追加を勧めるもの**: 利用規約・プライバシーポリシー(生年月日・性別・メールアドレスを扱うため)、通報・ブロック機能、退会機能
