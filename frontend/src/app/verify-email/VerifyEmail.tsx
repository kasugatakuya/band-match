"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState } from "react";
import useSWR from "swr";
import { AuthCard } from "@/components/AuthCard";
import { Button, ButtonLink, ErrorNote, Loading, Notice } from "@/components/ui";
import { api, errorMessage, useMe } from "@/lib/api";

export function VerifyEmail() {
  const token = useSearchParams().get("token");
  return token ? <ConfirmToken token={token} /> : <ResendPanel />;
}

/** メールのリンクから来たとき。トークンは1回しか使えないので、SWR に任せて二重送信を防ぐ。 */
function ConfirmToken({ token }: { token: string }) {
  const { mutate } = useMe();
  const { data, error } = useSWR(
    ["verify-email", token],
    async () => {
      await api("/auth/verify-email", { method: "POST", body: { token } });
      await mutate();
      return true;
    },
    { revalidateOnFocus: false, revalidateOnReconnect: false, revalidateIfStale: false, shouldRetryOnError: false },
  );

  if (error) {
    return (
      <AuthCard title="確認できませんでした">
        <div className="space-y-4">
          <ErrorNote>{errorMessage(error)}</ErrorNote>
          <ResendButton />
        </div>
      </AuthCard>
    );
  }
  if (!data) return <Loading label="確認しています…" />;
  return (
    <AuthCard title="メールアドレスを確認しました">
      <p className="mb-5 text-sm">記事の投稿とメッセージの送信ができるようになりました。</p>
      <ButtonLink href="/" className="w-full">記事を探す</ButtonLink>
    </AuthCard>
  );
}

function ResendPanel() {
  const sent = useSearchParams().get("sent") === "1";
  const { me, loading } = useMe();
  if (loading) return <Loading />;
  if (!me) {
    return (
      <AuthCard title="メールアドレスの確認">
        <p className="mb-5 text-sm">確認メールを再送するには、ログインしてください。</p>
        <ButtonLink href="/login?next=/verify-email" className="w-full">ログイン</ButtonLink>
      </AuthCard>
    );
  }
  if (me.email_verified) {
    return (
      <AuthCard title="確認済みです">
        <p className="mb-5 text-sm">メールアドレスは確認済みです。</p>
        <ButtonLink href="/" className="w-full">記事を探す</ButtonLink>
      </AuthCard>
    );
  }
  return (
    <AuthCard title={sent ? "確認メールを送りました" : "メールアドレスの確認"}>
      <div className="space-y-4 text-sm">
        <p>
          <strong>{me.email}</strong> 宛に確認メールを送りました。メール内のリンクを開くと登録が完了します(48時間有効)。
        </p>
        <p className="text-muted">届かない場合は、迷惑メールフォルダを確認するか、下のボタンから再送してください。</p>
        <ResendButton />
        <p className="text-center">
          <Link href="/" className="underline">あとで確認する</Link>
        </p>
      </div>
    </AuthCard>
  );
}

function ResendButton() {
  const { me } = useMe();
  const [state, setState] = useState<"idle" | "busy" | "done">("idle");
  const [error, setError] = useState("");
  if (!me) return <ButtonLink href="/login?next=/verify-email" variant="outline" className="w-full">ログインして再送する</ButtonLink>;
  if (state === "done") return <Notice>確認メールを再送しました。</Notice>;
  return (
    <>
      <ErrorNote>{error}</ErrorNote>
      <Button
        variant="outline"
        className="w-full"
        disabled={state === "busy"}
        onClick={async () => {
          setState("busy");
          setError("");
          try {
            await api("/auth/resend-verification", { method: "POST" });
            setState("done");
          } catch (err) {
            setError(errorMessage(err));
            setState("idle");
          }
        }}
      >
        確認メールを再送する
      </Button>
    </>
  );
}
