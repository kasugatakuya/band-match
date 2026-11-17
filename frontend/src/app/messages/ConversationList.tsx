"use client";

import Link from "next/link";
import useSWR from "swr";
import { Avatar, ButtonLink, ErrorNote, Loading, PageTitle } from "@/components/ui";
import { fetcher, type Conversation } from "@/lib/api";
import { useRequireAuth } from "@/lib/auth";
import { formatRelative } from "@/lib/format";

export function ConversationList() {
  const { me } = useRequireAuth();
  const { data, error } = useSWR<{ conversations: Conversation[] }>(me ? "/conversations" : null, fetcher, { refreshInterval: 15_000 });
  if (!me) return <Loading />;

  return (
    <div className="mx-auto max-w-2xl">
      <PageTitle>メッセージ</PageTitle>
      {error ? (
        <ErrorNote>メッセージを読み込めませんでした。</ErrorNote>
      ) : !data ? (
        <Loading />
      ) : data.conversations.length === 0 ? (
        <div className="rounded-lg border border-dashed border-line px-4 py-14 text-center">
          <p className="font-bold">まだメッセージはありません</p>
          <p className="mt-1 text-sm text-muted">気になる記事を見つけたら、記事のページからメッセージを送れます。</p>
          <ButtonLink href="/" variant="outline" className="mt-4">記事を探す</ButtonLink>
        </div>
      ) : (
        <ul className="divide-y divide-line overflow-hidden rounded-lg border border-line bg-card">
          {data.conversations.map((c) => {
            const unread = c.unread ?? 0;
            return (
              <li key={c.id}>
                <Link href={`/messages/${c.id}`} className="flex items-center gap-3 p-4 hover:bg-ink/[0.03] focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ink">
                  <Avatar name={c.other.display_name} url={c.other.avatar_url} size={44} />
                  <span className="min-w-0 flex-1">
                    <span className="flex items-baseline gap-2">
                      <span className={`truncate ${unread ? "font-black" : "font-bold"}`}>{c.other.display_name}</span>
                      {c.last_message_at && <span className="ml-auto shrink-0 text-xs text-muted">{formatRelative(c.last_message_at)}</span>}
                    </span>
                    {c.post_title && <span className="block truncate text-xs text-muted">{c.post_title}</span>}
                    <span className={`block truncate text-sm ${unread ? "font-bold text-ink" : "text-muted"}`}>{c.last_body}</span>
                  </span>
                  {unread > 0 && (
                    <span className="flex h-6 min-w-6 shrink-0 items-center justify-center rounded-full bg-accent px-1.5 text-xs font-bold text-white">
                      <span className="sr-only">未読</span>
                      {unread}
                    </span>
                  )}
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
