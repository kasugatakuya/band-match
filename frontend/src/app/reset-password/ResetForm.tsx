"use client";

import { useSearchParams } from "next/navigation";
import { useState } from "react";
import { AuthCard } from "@/components/AuthCard";
import { Button, ButtonLink, ErrorNote, Field } from "@/components/ui";
import { api, errorMessage, useMe } from "@/lib/api";

export function ResetForm() {
  const token = useSearchParams().get("token") ?? "";
  const { mutate } = useMe();
  const [password, setPassword] = useState("");
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  if (done) {
    return (
      <AuthCard title="パスワードを変更しました">
        <p className="mb-5 text-sm">新しいパスワードでログインしてください。</p>
        <ButtonLink href="/login" className="w-full">ログイン</ButtonLink>
      </AuthCard>
    );
  }
  if (!token) {
    return (
      <AuthCard title="リンクが正しくありません">
        <p className="mb-5 text-sm">メールのリンクをもう一度開くか、再設定メールを送り直してください。</p>
        <ButtonLink href="/forgot-password" variant="outline" className="w-full">再設定メールを送る</ButtonLink>
      </AuthCard>
    );
  }
  return (
    <AuthCard title="新しいパスワードの設定">
      <form
        className="space-y-4"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError("");
          try {
            await api("/auth/password/reset", { method: "POST", body: { token, password } });
            await mutate(); // 再設定すると全端末がログアウトされる
            setDone(true);
          } catch (err) {
            setError(errorMessage(err));
            setBusy(false);
          }
        }}
      >
        <Field label="新しいパスワード" hint="8文字以上">
          <input className="field" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} maxLength={72} />
        </Field>
        <ErrorNote>{error}</ErrorNote>
        <Button type="submit" className="w-full" disabled={busy}>パスワードを変更</Button>
      </form>
    </AuthCard>
  );
}
