"use client";

import Link from "next/link";
import { useState } from "react";
import { AuthCard } from "@/components/AuthCard";
import { Button, ErrorNote, Field } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";

export function ForgotForm() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  if (sent) {
    return (
      <AuthCard title="メールを送りました">
        <p className="text-sm">
          <strong>{email}</strong> が登録されている場合、パスワード再設定用のリンクをお送りしました(1時間有効)。メールをご確認ください。
        </p>
      </AuthCard>
    );
  }
  return (
    <AuthCard title="パスワードの再設定" footer={<Link href="/login" className="underline">ログインに戻る</Link>}>
      <form
        className="space-y-4"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError("");
          try {
            await api("/auth/password/forgot", { method: "POST", body: { email } });
            setSent(true);
          } catch (err) {
            setError(errorMessage(err));
            setBusy(false);
          }
        }}
      >
        <Field label="登録したメールアドレス" hint="Google で登録した方も、ここからパスワードを設定できます。">
          <input className="field" type="email" autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        </Field>
        <ErrorNote>{error}</ErrorNote>
        <Button type="submit" className="w-full" disabled={busy}>再設定メールを送る</Button>
      </form>
    </AuthCard>
  );
}
