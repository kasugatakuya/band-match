"use client";

import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { useMe } from "./api";

/**
 * ログインが必要なページで使う。未ログインならログイン画面へ、
 * プロフィール(生年月日・性別)が未入力なら入力画面へ移動する。
 * 表示してよい状態になったらユーザーを返す(それまでは null)。
 */
export function useRequireAuth(opts?: { allowIncompleteProfile?: boolean }) {
  const { me, loading, mutate } = useMe();
  const router = useRouter();
  const pathname = usePathname();
  const needsProfile = !!me && !me.profile_complete && !opts?.allowIncompleteProfile;

  useEffect(() => {
    if (loading) return;
    if (!me) router.replace(`/login?next=${encodeURIComponent(pathname)}`);
    else if (needsProfile) router.replace("/onboarding");
  }, [loading, me, needsProfile, pathname, router]);

  return { me: loading || needsProfile ? null : me, mutate };
}
