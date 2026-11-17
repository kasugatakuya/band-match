"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import useSWR, { useSWRConfig } from "swr";
import { PostCard } from "@/components/PostCard";
import { ProfileForm } from "@/components/ProfileForm";
import { Button, ButtonLink, Loading, PageTitle } from "@/components/ui";
import { api, fetcher, type Post } from "@/lib/api";
import { useRequireAuth } from "@/lib/auth";

export function MyPage() {
  const router = useRouter();
  const { me, mutate } = useRequireAuth();
  const { mutate: mutateAll } = useSWRConfig();
  const { data } = useSWR<{ posts: Post[] }>(me ? "/me/posts" : null, fetcher);
  if (!me) return <Loading />;

  const logout = async () => {
    await api("/auth/logout", { method: "POST" });
    router.replace("/");
    // 前のユーザーのデータが残らないよう、キャッシュをすべて捨てる
    await mutateAll(() => true, undefined, { revalidate: true });
  };

  return (
    <div className="grid gap-10 lg:grid-cols-2">
      <section>
        <PageTitle sub={<Link href={`/users/${me.id}`} className="underline">公開プロフィールを見る</Link>}>マイページ</PageTitle>
        <ProfileForm
          key={me.id}
          me={me}
          submitLabel="プロフィールを保存"
          onSaved={async (user) => {
            await mutate({ user }, { revalidate: false });
          }}
        />
        <div className="mt-10 border-t border-line pt-6">
          <h2 className="mb-2 text-sm font-bold">アカウント</h2>
          <p className="mb-4 text-sm text-muted">
            {me.email}
            {me.google_linked && "・Google ログイン連携済み"}
            {!me.has_password && (
              <>
                ・<Link href="/forgot-password" className="underline">パスワードを設定する</Link>
              </>
            )}
          </p>
          <Button variant="outline" onClick={logout}>ログアウト</Button>
        </div>
      </section>

      <section>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-xl font-black">自分の記事</h2>
          <ButtonLink href="/posts/new">記事を書く</ButtonLink>
        </div>
        {!data ? (
          <Loading />
        ) : data.posts.length === 0 ? (
          <p className="rounded-lg border border-dashed border-line px-4 py-10 text-center text-sm text-muted">まだ記事がありません。</p>
        ) : (
          <ul className="space-y-4">
            {data.posts.map((post) => (
              <li key={post.id}>
                <PostCard post={post} showStatus />
                <p className="mt-1.5 text-right text-sm">
                  <Link href={`/posts/${post.id}/edit`} className="font-bold underline">編集・締め切り</Link>
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
