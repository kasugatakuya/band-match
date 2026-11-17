"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import useSWR from "swr";
import { AuthCard, Divider, GoogleButton } from "@/components/AuthCard";
import { Button, ErrorNote, Field } from "@/components/ui";
import { api, errorMessage, fetcher, useMe, type Me } from "@/lib/api";
import { safeNext } from "@/lib/format";

const GOOGLE_ERRORS: Record<string, string> = {
  google_unavailable: "Google ログインは現在利用できません。メールアドレスでログインしてください。",
  google_failed: "Google ログインに失敗しました。もう一度お試しください。",
  google_email_unverified: "Google アカウントのメールアドレスが確認されていないため、ログインできませんでした。",
};

export function LoginForm() {
  const router = useRouter();
  const params = useSearchParams();
  const next = safeNext(params.get("next"));
  const { mutate } = useMe();
  const { data: config } = useSWR<{ google_enabled: boolean; dev_login: boolean }>("/auth/config", fetcher, { revalidateOnFocus: false });
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState(GOOGLE_ERRORS[params.get("error") ?? ""] ?? "");
  const [busy, setBusy] = useState(false);

  const finish = async (user: Me) => {
    await mutate({ user }, { revalidate: false });
    router.replace(user.profile_complete ? next : "/onboarding");
  };

  const run = async (path: string, body: unknown) => {
    setBusy(true);
    setError("");
    try {
      const res = await api<{ user: Me }>(path, { method: "POST", body });
      await finish(res.user);
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <AuthCard
      title="ログイン"
      footer={<>アカウントをお持ちでない方は <Link href="/signup" className="font-bold text-ink underline">新規登録</Link></>}
    >
      {config?.google_enabled && (
        <>
          <GoogleButton label="Google でログイン" />
          <Divider />
        </>
      )}
      <form
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault();
          run("/auth/login", { email, password });
        }}
      >
        <Field label="メールアドレス">
          <input className="field" type="email" autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        </Field>
        <Field label="パスワード">
          <input className="field" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        </Field>
        <ErrorNote>{error}</ErrorNote>
        <Button type="submit" className="w-full" disabled={busy}>{busy ? "ログイン中…" : "ログイン"}</Button>
        <p className="text-center text-sm">
          <Link href="/forgot-password" className="underline">パスワードを忘れた方</Link>
        </p>
      </form>

      {config?.dev_login && (
        <div className="mt-6 rounded-md border border-dashed border-ink/40 p-3">
          <p className="mb-2 text-xs font-bold">開発用ログイン(本番では表示されません)</p>
          <div className="flex flex-wrap gap-2">
            {[
              { email: "taro@example.com", display_name: "テスト太郎" },
              { email: "hanako@example.com", display_name: "テスト花子" },
            ].map((u) => (
              <Button key={u.email} type="button" variant="outline" disabled={busy} onClick={() => run("/auth/dev-login", u)}>
                {u.display_name}
              </Button>
            ))}
          </div>
        </div>
      )}
    </AuthCard>
  );
}
