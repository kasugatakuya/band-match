import Link from "next/link";
import type { Post } from "@/lib/api";
import { genderLabel } from "@/lib/constants";
import { formatRelative } from "@/lib/format";
import { Avatar, Chip, KindBadge } from "./ui";

export function authorMeta(a: { age: number | null; gender: string | null }): string {
  const gender = a.gender && a.gender !== "no_answer" ? genderLabel(a.gender) : "";
  return [a.age != null ? `${a.age}歳` : "", gender].filter(Boolean).join("・");
}

export function PostCard({ post, showStatus = false }: { post: Post; showStatus?: boolean }) {
  return (
    <article className="relative rounded-lg border border-line bg-card p-4 transition-shadow hover:shadow-[4px_4px_0_0_var(--color-ink)] sm:p-5">
      <div className="mb-2 flex flex-wrap items-center gap-2 text-xs text-muted">
        <KindBadge kind={post.kind} />
        {showStatus && post.status === "closed" && <span className="rounded-sm bg-muted px-2 py-1 font-bold leading-none text-white">締め切り</span>}
        <span>{post.prefecture}</span>
        <span className="ml-auto">{formatRelative(post.created_at)}</span>
      </div>
      <h2 className="text-lg font-black leading-snug">
        <Link href={`/posts/${post.id}`} className="after:absolute after:inset-0 focus-visible:outline-none focus-visible:after:rounded-lg focus-visible:after:outline-2 focus-visible:after:outline-ink">
          {post.title}
        </Link>
      </h2>
      <p className="mt-1.5 line-clamp-2 text-sm text-muted">{post.body}</p>
      <div className="mt-3 flex flex-wrap gap-1.5">
        {post.parts.map((p) => (
          <span key={p} className="inline-block rounded-full bg-ink px-2.5 py-1 text-xs font-bold leading-none text-white">{p}</span>
        ))}
        {post.genres.map((g) => <Chip key={g}>{g}</Chip>)}
      </div>
      {post.artists.length > 0 && <p className="mt-2 truncate text-xs text-muted">好きなアーティスト: {post.artists.join(" / ")}</p>}
      <div className="mt-3 flex items-center gap-2 border-t border-line pt-3 text-sm">
        <Avatar name={post.author.display_name} url={post.author.avatar_url} size={24} />
        <span className="truncate font-bold">{post.author.display_name}</span>
        <span className="shrink-0 text-xs text-muted">{authorMeta(post.author)}</span>
      </div>
    </article>
  );
}
