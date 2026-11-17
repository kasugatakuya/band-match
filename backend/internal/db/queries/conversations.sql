-- name: GetConversationByPostAndStarter :one
SELECT * FROM conversations WHERE post_id = $1 AND starter_id = $2;

-- name: CreateConversation :one
INSERT INTO conversations (post_id, starter_id, owner_id) VALUES ($1, $2, $3)
RETURNING *;

-- name: AddConversationMember :exec
INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1, $2);

-- name: GetConversationForUser :one
SELECT c.id, c.post_id, c.last_message_at,
       COALESCE(p.title, '')::text AS post_title,
       COALESCE(p.kind, '')::text AS post_kind,
       ou.id AS other_id, ou.display_name AS other_name, ou.avatar_url AS other_avatar,
       ou.email AS other_email, ou.email_notifications AS other_email_notifications
FROM conversation_members me
JOIN conversations c ON c.id = me.conversation_id
LEFT JOIN posts p ON p.id = c.post_id
JOIN users ou ON ou.id = CASE WHEN c.starter_id = me.user_id THEN c.owner_id ELSE c.starter_id END
WHERE me.conversation_id = $1 AND me.user_id = $2;

-- name: ListConversationsForUser :many
SELECT c.id, c.post_id, c.last_message_at,
       COALESCE(p.title, '')::text AS post_title,
       COALESCE(p.kind, '')::text AS post_kind,
       ou.id AS other_id, ou.display_name AS other_name, ou.avatar_url AS other_avatar,
       COALESCE((SELECT m.body FROM messages m WHERE m.conversation_id = c.id
                 ORDER BY m.id DESC LIMIT 1), '')::text AS last_body,
       (SELECT count(*) FROM messages m
         WHERE m.conversation_id = c.id AND m.sender_id <> me.user_id
           AND m.id > me.last_read_message_id)::bigint AS unread
FROM conversation_members me
JOIN conversations c ON c.id = me.conversation_id
LEFT JOIN posts p ON p.id = c.post_id
JOIN users ou ON ou.id = CASE WHEN c.starter_id = me.user_id THEN c.owner_id ELSE c.starter_id END
WHERE me.user_id = $1
ORDER BY c.last_message_at DESC
LIMIT 200;

-- name: CountUnreadForUser :one
SELECT count(*)::bigint
FROM conversation_members me
JOIN messages m ON m.conversation_id = me.conversation_id
WHERE me.user_id = $1 AND m.sender_id <> me.user_id AND m.id > me.last_read_message_id;

-- name: ListMessages :many
SELECT * FROM messages
WHERE conversation_id = $1 AND id > sqlc.arg(after_id)
ORDER BY id DESC
LIMIT 300;

-- name: CreateMessage :one
INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
RETURNING *;

-- name: TouchConversation :exec
UPDATE conversations SET last_message_at = now() WHERE id = $1;

-- name: MarkConversationRead :exec
UPDATE conversation_members cm
SET last_read_message_id = GREATEST(cm.last_read_message_id,
        (SELECT COALESCE(max(m.id), 0) FROM messages m WHERE m.conversation_id = cm.conversation_id)),
    notify_pending = false
WHERE cm.conversation_id = $1 AND cm.user_id = $2;

-- name: ClaimNotification :execrows
UPDATE conversation_members SET notify_pending = true
WHERE conversation_id = $1 AND user_id = $2 AND notify_pending = false;

-- name: CountRecentMessagesBySender :one
SELECT count(*) FROM messages WHERE sender_id = $1 AND created_at > now() - interval '1 minute';

-- name: CountRecentConversationsByStarter :one
SELECT count(*) FROM conversations WHERE starter_id = $1 AND created_at > now() - interval '1 day';

-- name: EnqueueEmail :one
INSERT INTO email_outbox (to_email, subject, body) VALUES ($1, $2, $3)
RETURNING id;

-- name: MarkEmailSent :exec
UPDATE email_outbox SET status = 'sent', sent_at = now(), attempts = attempts + 1 WHERE id = $1;

-- name: MarkEmailFailed :exec
UPDATE email_outbox
SET attempts = attempts + 1,
    status = CASE WHEN attempts + 1 >= 5 THEN 'failed' ELSE 'pending' END
WHERE id = $1;

-- name: ListRetryableEmails :many
SELECT * FROM email_outbox
WHERE status = 'pending' AND created_at < now() - interval '1 minute'
ORDER BY id
LIMIT 20;
