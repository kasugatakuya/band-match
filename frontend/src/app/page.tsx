import { Suspense } from "react";
import { PostList } from "@/components/PostList";
import { ButtonLink, Loading } from "@/components/ui";

export default function Home() {
  return (
    <>
      <section className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-3xl font-black leading-tight tracking-tight sm:text-4xl">
            一緒に音を出す仲間を、<br className="sm:hidden" />見つけよう。
          </h1>
          <p className="mt-2 text-sm text-muted">メンバー募集と加入希望の記事を、地域・パート・ジャンル・好きなアーティストで探せます。</p>
        </div>
        {/* PC 幅ではヘッダーに同じボタンがあるので、スマホ幅だけ出す */}
        <div className="sm:hidden"><ButtonLink href="/posts/new">記事を書く</ButtonLink></div>
      </section>
      <Suspense fallback={<Loading />}>
        <PostList />
      </Suspense>
    </>
  );
}
