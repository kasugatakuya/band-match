"use client";

import Link from "next/link";
import useSWR from "swr";
import { fetcher, useMe } from "@/lib/api";
import { Avatar, ButtonLink } from "./ui";

export function Header() {
  const { me, loading } = useMe();
  // 未読数は30秒ごとに取り直す
  const { data } = useSWR<{ unread: number }>(me ? "/conversations/unread-count" : null, fetcher, { refreshInterval: 30_000 });
  const unread = data?.unread ?? 0;

  return (
    <header className="sticky top-0 z-20 border-b border-line bg-paper/95 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-5xl items-center gap-2 px-4">
        <Link href="/" className="mr-auto flex items-center gap-2 text-lg font-black tracking-tight">
          <span aria-hidden className="flex h-7 w-7 items-center justify-center rounded-full bg-accent text-sm text-white">♪</span>
          BandMatch
        </Link>
        {loading ? null : me ? (
          <>
            <Link href="/messages" className="relative whitespace-nowrap rounded-md px-3 py-2 text-sm font-bold hover:bg-ink/5">
              メッセージ
              {unread > 0 && (
                <span className="absolute -right-0.5 -top-0.5 flex h-5 min-w-5 items-center justify-center rounded-full bg-accent px-1 text-[11px] font-bold leading-none text-white">
                  <span className="sr-only">未読</span>
                  {unread > 99 ? "99+" : unread}
                </span>
              )}
            </Link>
            {/* スマホ幅ではヘッダーが窮屈になるので隠す(トップページとフッターから行ける) */}
            <span className="hidden sm:block"><ButtonLink href="/posts/new">記事を書く</ButtonLink></span>
            <Link href="/mypage" aria-label="マイページ" className="ml-1 rounded-full focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink">
              <Avatar name={me.display_name} url={me.avatar_url} size={34} />
            </Link>
          </>
        ) : (
          <>
            <ButtonLink href="/login" variant="ghost">ログイン</ButtonLink>
            <ButtonLink href="/signup" variant="dark">新規登録</ButtonLink>
          </>
        )}
      </div>
    </header>
  );
}

/** メール未確認のときに全ページ上部へ出す案内。 */
export function VerifyBanner() {
  const { me } = useMe();
  if (!me || me.email_verified) return null;
  return (
    <div className="border-b border-sun/60 bg-sun/20">
      <p className="mx-auto max-w-5xl px-4 py-2.5 text-sm">
        メールアドレスの確認が済んでいません。届いたメールのリンクを開くと、記事の投稿とメッセージ送信ができるようになります。{" "}
        <Link href="/verify-email" className="font-bold underline">確認メールを再送する</Link>
      </p>
    </div>
  );
}
