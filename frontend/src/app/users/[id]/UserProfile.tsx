"use client";

import { useParams } from "next/navigation";
import useSWR from "swr";
import { authorMeta, PostCard } from "@/components/PostCard";
import { Avatar, ButtonLink, Chip, ErrorNote, Loading } from "@/components/ui";
import { fetcher, useMe, type Post, type PublicUser } from "@/lib/api";
import { formatDate } from "@/lib/format";

export function UserProfile() {
  const { id } = useParams<{ id: string }>();
  const { me } = useMe();
  const { data, error } = useSWR<{ user: PublicUser; posts: Post[] }>(`/users/${id}`, fetcher);
  if (error) return <ErrorNote>ユーザーが見つかりませんでした。</ErrorNote>;
  if (!data) return <Loading />;
  const { user, posts } = data;

  return (
    <div className="mx-auto max-w-3xl">
      <title>{`${user.display_name} | BandMatch`}</title>
      <div className="flex items-center gap-4">
        <Avatar name={user.display_name} url={user.avatar_url} size={72} />
        <div className="min-w-0">
          <h1 className="truncate text-2xl font-black tracking-tight">{user.display_name}</h1>
          <p className="mt-1 text-sm text-muted">
            {[authorMeta(user), user.prefecture, `${formatDate(user.created_at)}に登録`].filter(Boolean).join("・")}
          </p>
        </div>
        {me?.id === user.id && <ButtonLink href="/mypage" variant="outline" className="ml-auto shrink-0">編集</ButtonLink>}
      </div>
      {user.parts.length > 0 && <div className="mt-4 flex flex-wrap gap-1.5">{user.parts.map((p) => <Chip key={p}>{p}</Chip>)}</div>}
      {user.bio && <p className="mt-4 whitespace-pre-wrap break-words leading-7">{user.bio}</p>}

      <h2 className="mb-4 mt-10 text-xl font-black">募集中の記事</h2>
      {posts.length === 0 ? (
        <p className="rounded-lg border border-dashed border-line px-4 py-10 text-center text-sm text-muted">募集中の記事はありません。</p>
      ) : (
        <ul className="grid gap-4 md:grid-cols-2">
          {posts.map((post) => <li key={post.id}><PostCard post={post} /></li>)}
        </ul>
      )}
    </div>
  );
}
