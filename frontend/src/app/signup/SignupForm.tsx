"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import useSWR from "swr";
import { AuthCard, Divider, GoogleButton } from "@/components/AuthCard";
import { Button, ErrorNote, Field } from "@/components/ui";
import { api, errorMessage, fetcher, useMe, type Me } from "@/lib/api";
import { GENDERS } from "@/lib/constants";

export function SignupForm() {
  const router = useRouter();
  const { mutate } = useMe();
  const { data: config } = useSWR<{ google_enabled: boolean }>("/auth/config", fetcher, { revalidateOnFocus: false });
  const [form, setForm] = useState({ email: "", password: "", display_name: "", birth_date: "", gender: "" });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const set = (key: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setForm({ ...form, [key]: e.target.value });

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const res = await api<{ user: Me }>("/auth/signup", { method: "POST", body: form });
      await mutate({ user: res.user }, { revalidate: false });
      router.replace("/verify-email?sent=1");
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <AuthCard
      title="新規登録"
      footer={<>すでにアカウントをお持ちの方は <Link href="/login" className="font-bold text-ink underline">ログイン</Link></>}
    >
      {config?.google_enabled && (
        <>
          <GoogleButton label="Google で登録" />
          <Divider />
        </>
      )}
      <form onSubmit={submit} className="space-y-4">
        <Field label="メールアドレス" required hint="メッセージが届いたときの通知先になります。他のユーザーには公開されません。">
          <input className="field" type="email" autoComplete="email" value={form.email} onChange={set("email")} required />
        </Field>
        <Field label="パスワード" required hint="8文字以上">
          <input className="field" type="password" autoComplete="new-password" value={form.password} onChange={set("password")} required minLength={8} maxLength={72} />
        </Field>
        <Field label="表示名" required hint="記事やメッセージに表示される名前です。">
          <input className="field" autoComplete="nickname" value={form.display_name} onChange={set("display_name")} required maxLength={30} />
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="生年月日" required hint="公開されるのは年齢だけです。">
            <input className="field" type="date" autoComplete="bday" value={form.birth_date} onChange={set("birth_date")} required />
          </Field>
          <Field label="性別" required>
            <select className="field" value={form.gender} onChange={set("gender")} required>
              <option value="" disabled>選択してください</option>
              {GENDERS.map((g) => <option key={g.value} value={g.value}>{g.label}</option>)}
            </select>
          </Field>
        </div>
        <ErrorNote>{error}</ErrorNote>
        <Button type="submit" className="w-full" disabled={busy}>{busy ? "登録中…" : "登録する"}</Button>
      </form>
    </AuthCard>
  );
}
