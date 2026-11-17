-- +goose Up
CREATE TABLE users (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email               text NOT NULL UNIQUE, -- 小文字に正規化して保存する
    email_verified_at   timestamptz,
    password_hash       text,
    google_sub          text UNIQUE,
    display_name        text NOT NULL,
    -- 生年月日と性別は必須項目だが、Google ログイン直後は未入力なので NULL を許す。
    -- 両方そろうまで投稿・メッセージ送信はできない(アプリ側で制御)
    birth_date          date,
    gender              text CHECK (gender IN ('male', 'female', 'other', 'no_answer')),
    avatar_url          text NOT NULL DEFAULT '',
    bio                 text NOT NULL DEFAULT '',
    prefecture          text NOT NULL DEFAULT '',
    parts               text[] NOT NULL DEFAULT '{}',
    email_notifications boolean NOT NULL DEFAULT true,
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);

CREATE TABLE email_tokens (
    token_hash bytea PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('verify', 'reset')),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX email_tokens_user_id_idx ON email_tokens (user_id);

CREATE TABLE posts (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('recruit', 'join')),
    title      text NOT NULL,
    body       text NOT NULL,
    prefecture text NOT NULL,
    parts      text[] NOT NULL DEFAULT '{}',
    genres     text[] NOT NULL DEFAULT '{}',
    artists    text[] NOT NULL DEFAULT '{}',
    status     text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX posts_list_idx ON posts (status, kind, created_at DESC);
CREATE INDEX posts_user_id_idx ON posts (user_id);
CREATE INDEX posts_parts_idx ON posts USING gin (parts);
CREATE INDEX posts_genres_idx ON posts USING gin (genres);

CREATE TABLE conversations (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id         uuid REFERENCES posts (id) ON DELETE SET NULL, -- 記事が消えても会話は残す
    starter_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    owner_id        uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at      timestamptz NOT NULL DEFAULT now(),
    last_message_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (post_id, starter_id)
);

CREATE TABLE conversation_members (
    conversation_id      uuid NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    user_id              uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    last_read_message_id bigint NOT NULL DEFAULT 0,
    -- true の間は「未読あり」の通知メールを送信済み。既読にすると false に戻る
    notify_pending       boolean NOT NULL DEFAULT false,
    PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX conversation_members_user_id_idx ON conversation_members (user_id);

CREATE TABLE messages (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    conversation_id uuid NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    sender_id       uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body            text NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX messages_conversation_idx ON messages (conversation_id, id);
CREATE INDEX messages_sender_idx ON messages (sender_id, created_at);

CREATE TABLE email_outbox (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    to_email   text NOT NULL,
    subject    text NOT NULL,
    body       text NOT NULL,
    status     text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    attempts   integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    sent_at    timestamptz
);
CREATE INDEX email_outbox_pending_idx ON email_outbox (created_at) WHERE status = 'pending';

-- +goose Down
DROP TABLE email_outbox;
DROP TABLE messages;
DROP TABLE conversation_members;
DROP TABLE conversations;
DROP TABLE posts;
DROP TABLE email_tokens;
DROP TABLE sessions;
DROP TABLE users;
