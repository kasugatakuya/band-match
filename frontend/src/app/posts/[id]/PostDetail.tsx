"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";
import useSWR from "swr";
import { authorMeta } from "@/components/PostCard";
import { Avatar, Button, ButtonLink, Chip, ErrorNote, KindBadge, Loading, Notice } from "@/components/ui";
import { api, ApiError, errorMessage, fetcher, useMe, type Post } from "@/lib/api";
import { formatDate } from "@/lib/format";

export function PostDetail() {
  const { id } = useParams<{ id: string }>();
  const { me, loading: meLoading } = useMe();
  const { data, error } = useSWR<{ post: Post }>(`/posts/${id}`, fetcher);

  if (error) {
    return (
      <div className="py-16 text-center">
        <p className="font-bold">{error instanceof ApiError && error.status === 404 ? "この記事は見つかりませんでした。削除された可能性があります。" : "記事を読み込めませんでした。"}</p>
        <ButtonLink href="/" variant="outline" className="mt-4">記事一覧へ</ButtonLink>
      </div>
    );
  }
  if (!data) return <Loading />;
  const { post } = data;
  const isOwner = me?.id === post.author.id;

  return (
    <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_320px]">
      <article>
        <title>{`${post.title} | BandMatch`}</title>
        <div className="mb-3 flex flex-wrap items-center gap-2 text-sm text-muted">
          <KindBadge kind={post.kind} />
          {post.status === "closed" && <span className="rounded-sm bg-muted px-2 py-1 text-xs font-bold leading-none text-white">締め切り</span>}
          <span>{post.prefecture}</span>
          <span>{formatDate(post.created_at)}</span>
        </div>
        <h1 className="text-2xl font-black leading-snug tracking-tight sm:text-3xl">{post.title}</h1>

        <dl className="mt-6 space-y-4 rounded-lg border border-line bg-card p-4 text-sm sm:p-5">
          <div>
            <dt className="mb-1.5 text-xs font-bold text-muted">{post.kind === "recruit" ? "募集しているパート" : "担当パート"}</dt>
            <dd className="flex flex-wrap gap-1.5">
              {post.parts.map((p) => (
                <span key={p} className="inline-block rounded-full bg-ink px-3 py-1.5 text-sm font-bold leading-none text-white">{p}</span>
              ))}
            </dd>
          </div>
          {post.genres.length > 0 && (
            <div>
              <dt className="mb-1.5 text-xs font-bold text-muted">ジャンル</dt>
              <dd className="flex flex-wrap gap-1.5">{post.genres.map((g) => <Chip key={g}>{g}</Chip>)}</dd>
            </div>
          )}
          {post.artists.length > 0 && (
            <div>
              <dt className="mb-1.5 text-xs font-bold text-muted">好きなアーティスト</dt>
              <dd className="flex flex-wrap gap-1.5">
                {post.artists.map((a) => (
                  <Link key={a} href={`/?artist=${encodeURIComponent(a)}`} className="rounded-full border border-line bg-white px-2.5 py-1 text-xs leading-none hover:border-ink">
                    {a}
                  </Link>
                ))}
              </dd>
            </div>
          )}
        </dl>

        <p className="mt-6 whitespace-pre-wrap break-words leading-8">{post.body}</p>
      </article>

      <aside className="space-y-4 lg:sticky lg:top-20 lg:self-start">
        <Link href={`/users/${post.author.id}`} className="flex items-center gap-3 rounded-lg border border-line bg-card p-4 hover:border-ink">
          <Avatar name={post.author.display_name} url={post.author.avatar_url} size={48} />
          <span className="min-w-0">
            <span className="block truncate font-black">{post.author.display_name}</span>
            <span className="block text-xs text-muted">{authorMeta(post.author) || "プロフィールを見る"}</span>
          </span>
        </Link>

        {meLoading ? null : isOwner ? (
          <div className="rounded-lg border border-line bg-card p-4">
            <p className="mb-3 text-sm text-muted">あなたの記事です。届いたメッセージは「メッセージ」から確認できます。</p>
            <ButtonLink href={`/posts/${post.id}/edit`} variant="outline" className="w-full">編集・締め切り</ButtonLink>
          </div>
        ) : post.status === "closed" ? (
          <Notice>この記事は募集を締め切っています。</Notice>
        ) : !me ? (
          <div className="rounded-lg border border-line bg-card p-4">
            <p className="mb-3 text-sm">メッセージを送るにはログインが必要です。</p>
            <ButtonLink href={`/login?next=${encodeURIComponent(`/posts/${post.id}`)}`} className="w-full">ログインしてメッセージを送る</ButtonLink>
          </div>
        ) : !me.email_verified ? (
          <Notice>
            メッセージを送るには、メールアドレスの確認が必要です。<Link href="/verify-email" className="font-bold underline">確認メールを再送する</Link>
          </Notice>
        ) : !me.profile_complete ? (
          <Notice>
            メッセージを送るには、<Link href="/onboarding" className="font-bold underline">プロフィールの入力</Link>が必要です。
          </Notice>
        ) : (
          <MessageBox post={post} />
        )}
      </aside>
    </div>
  );
}

function MessageBox({ post }: { post: Post }) {
  const router = useRouter();
  const [body, setBody] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const send = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const res = await api<{ conversation_id: string }>(`/posts/${post.id}/conversations`, { method: "POST", body: { body } });
      router.push(`/messages/${res.conversation_id}`);
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <form onSubmit={send} className="space-y-3 rounded-lg border-2 border-ink bg-card p-4">
      <label className="block">
        <span className="mb-1.5 block font-black">{post.author.display_name} さんにメッセージを送る</span>
        <textarea
          className="field min-h-32"
          value={body}
          onChange={(e) => setBody(e.target.value)}
          required
          maxLength={2000}
          placeholder={post.kind === "recruit" ? "自己紹介、担当パート、経験、活動できる曜日などを書いてみましょう。" : "バンドの紹介、活動頻度、音源の有無などを書いてみましょう。"}
        />
      </label>
      <ErrorNote>{error}</ErrorNote>
      <Button type="submit" className="w-full" disabled={busy || !body.trim()}>{busy ? "送信中…" : "メッセージを送る"}</Button>
      <p className="text-xs text-muted">相手にメールで通知が届きます。送信後はメッセージ画面で、練習日程などの相談を続けられます。</p>
    </form>
  );
}
