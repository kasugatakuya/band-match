"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import useSWR, { useSWRConfig } from "swr";
import { Avatar, Button, ButtonLink, ErrorNote, Loading } from "@/components/ui";
import { api, errorMessage, fetcher, type Conversation, type Me, type Message } from "@/lib/api";
import { useRequireAuth } from "@/lib/auth";
import { formatDay, formatTime } from "@/lib/format";

const POLL_MS = 5000;

export function Chat() {
  const { id } = useParams<{ id: string }>();
  const { me } = useRequireAuth();
  const { data, error } = useSWR<{ conversation: Conversation }>(me ? `/conversations/${id}` : null, fetcher, { revalidateOnFocus: false });

  if (error) {
    return (
      <div className="py-16 text-center">
        <p className="font-bold">このメッセージは見つかりませんでした。</p>
        <ButtonLink href="/messages" variant="outline" className="mt-4">メッセージ一覧へ</ButtonLink>
      </div>
    );
  }
  if (!me || !data) return <Loading />;
  return <Thread key={id} me={me} conversation={data.conversation} />;
}

function Thread({ me, conversation }: { me: Me; conversation: Conversation }) {
  const { mutate } = useSWRConfig();
  const [messages, setMessages] = useState<Message[] | null>(null);
  const [loadError, setLoadError] = useState(false);
  const [draft, setDraft] = useState("");
  const [sendError, setSendError] = useState("");
  const [sending, setSending] = useState(false);
  const lastIdRef = useRef(0);
  const bottomRef = useRef<HTMLDivElement>(null);
  const base = `/conversations/${conversation.id}`;

  // 新着を取り込む。重複を避けるため id で判定する
  const append = (incoming: Message[]) => {
    const fresh = incoming.filter((m) => m.id > lastIdRef.current);
    if (fresh.length > 0) lastIdRef.current = fresh[fresh.length - 1].id;
    setMessages((prev) => (prev === null ? fresh : fresh.length > 0 ? [...prev, ...fresh] : prev));
    return fresh;
  };

  // 5秒ごとに新着を確認する。相手からの新着があれば既読にして未読バッジを更新する
  useEffect(() => {
    let active = true;
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      try {
        const res = await api<{ messages: Message[] }>(`${base}/messages?after=${lastIdRef.current}`);
        if (!active) return;
        const first = lastIdRef.current === 0;
        const fresh = append(res.messages);
        setLoadError(false);
        if (first || fresh.some((m) => m.sender_id !== me.id)) {
          await api(`${base}/read`, { method: "POST" });
          mutate("/conversations/unread-count");
          mutate("/conversations");
        }
      } catch {
        if (active) setLoadError(true);
      }
      if (active) timer = setTimeout(poll, POLL_MS);
    };
    poll();
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [base, me.id, mutate]);

  // メッセージが増えたら最新まで自動でスクロールする
  const count = messages?.length ?? 0;
  useLayoutEffect(() => {
    if (count > 0) bottomRef.current?.scrollIntoView({ block: "end" });
  }, [count]);

  const send = async () => {
    const body = draft.trim();
    if (!body || sending) return;
    setSending(true);
    setSendError("");
    try {
      const res = await api<{ message: Message }>(`${base}/messages`, { method: "POST", body: { body } });
      append([res.message]);
      setDraft("");
      mutate("/conversations");
    } catch (err) {
      setSendError(errorMessage(err));
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="mx-auto flex max-w-2xl flex-col">
      <div className="mb-4 flex items-center gap-3 border-b border-line pb-4">
        <Link href="/messages" aria-label="メッセージ一覧へ戻る" className="-ml-2 rounded-md px-2 py-1 text-xl hover:bg-ink/5">←</Link>
        <Avatar name={conversation.other.display_name} url={conversation.other.avatar_url} size={40} />
        <div className="min-w-0">
          <Link href={`/users/${conversation.other.id}`} className="block truncate font-black hover:underline">{conversation.other.display_name}</Link>
          {conversation.post_id ? (
            <Link href={`/posts/${conversation.post_id}`} className="block truncate text-xs text-muted hover:underline">記事: {conversation.post_title}</Link>
          ) : (
            <span className="block text-xs text-muted">元の記事は削除されました</span>
          )}
        </div>
      </div>

      {messages === null ? (
        loadError ? <ErrorNote>メッセージを読み込めませんでした。再読み込みしてください。</ErrorNote> : <Loading />
      ) : (
        <ol className="space-y-2 pb-4" aria-live="polite">
          {messages.map((m, i) => {
            const mine = m.sender_id === me.id;
            const day = formatDay(m.created_at);
            const showDay = i === 0 || formatDay(messages[i - 1].created_at) !== day;
            return (
              <li key={m.id}>
                {showDay && <p className="my-4 text-center text-xs text-muted">{day}</p>}
                <div className={`flex items-end gap-2 ${mine ? "flex-row-reverse" : ""}`}>
                  <p className={`max-w-[80%] whitespace-pre-wrap break-words rounded-2xl px-4 py-2.5 leading-7 ${mine ? "rounded-br-sm bg-ink text-white" : "rounded-bl-sm border border-line bg-white"}`}>
                    <span className="sr-only">{mine ? "自分: " : `${conversation.other.display_name}: `}</span>
                    {m.body}
                  </p>
                  <time dateTime={m.created_at} className="shrink-0 pb-1 text-[11px] text-muted">{formatTime(m.created_at)}</time>
                </div>
              </li>
            );
          })}
        </ol>
      )}
      <div ref={bottomRef} className="scroll-mb-40" />

      <form
        className="sticky bottom-0 -mx-4 border-t border-line bg-paper px-4 pb-4 pt-3"
        onSubmit={(e) => {
          e.preventDefault();
          send();
        }}
      >
        {loadError && messages !== null && <p className="mb-2 text-xs text-accent-dark">接続が不安定です。新着の確認を再試行しています…</p>}
        {sendError && <div className="mb-2"><ErrorNote>{sendError}</ErrorNote></div>}
        <div className="flex items-end gap-2">
          <label className="flex-1">
            <span className="sr-only">メッセージ</span>
            <textarea
              className="field max-h-40 min-h-12 resize-none"
              rows={2}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              maxLength={2000}
              placeholder="練習の候補日や場所などを相談しましょう"
              onKeyDown={(e) => {
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey) && !e.nativeEvent.isComposing) {
                  e.preventDefault();
                  send();
                }
              }}
            />
          </label>
          <Button type="submit" variant="primary" className="h-12" disabled={sending || !draft.trim()}>送信</Button>
        </div>
        <p className="mt-1.5 text-xs text-muted">⌘ / Ctrl + Enter でも送信できます。相手が未読の間は、最初の1通だけメールで通知されます。</p>
      </form>
    </div>
  );
}
