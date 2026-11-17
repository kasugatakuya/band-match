"use client";

import { useParams } from "next/navigation";
import useSWR from "swr";
import { PostForm } from "@/components/PostForm";
import { ErrorNote, Loading, PageTitle } from "@/components/ui";
import { fetcher, type Post } from "@/lib/api";
import { useRequireAuth } from "@/lib/auth";

export function EditPost() {
  const { id } = useParams<{ id: string }>();
  const { me } = useRequireAuth();
  const { data, error } = useSWR<{ post: Post }>(`/posts/${id}`, fetcher, { revalidateOnFocus: false });

  if (error) return <ErrorNote>記事が見つかりませんでした。</ErrorNote>;
  if (!me || !data) return <Loading />;
  if (data.post.author.id !== me.id) return <ErrorNote>この記事を編集する権限がありません。</ErrorNote>;
  return (
    <div className="mx-auto max-w-2xl">
      <PageTitle>記事を編集</PageTitle>
      <PostForm key={data.post.updated_at} post={data.post} />
    </div>
  );
}
