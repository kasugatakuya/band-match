"use client";

import Link from "next/link";
import { PostForm } from "@/components/PostForm";
import { Loading, Notice, PageTitle } from "@/components/ui";
import { useRequireAuth } from "@/lib/auth";

export function NewPost() {
  const { me } = useRequireAuth();
  if (!me) return <Loading />;
  return (
    <div className="mx-auto max-w-2xl">
      <PageTitle>記事を書く</PageTitle>
      {me.email_verified ? (
        <PostForm defaultPrefecture={me.prefecture} />
      ) : (
        <Notice>
          記事を投稿するには、メールアドレスの確認が必要です。<Link href="/verify-email" className="font-bold underline">確認メールを再送する</Link>
        </Notice>
      )}
    </div>
  );
}
