"use client";

import { useState } from "react";
import { api, errorMessage, type Me } from "@/lib/api";
import { GENDERS, PARTS, PREFECTURES } from "@/lib/constants";
import { ChipSelect } from "./inputs";
import { Button, ErrorNote, Field } from "./ui";

/** プロフィール編集。初回入力(onboarding)とマイページの両方で使う。 */
export function ProfileForm({ me, submitLabel, onSaved }: { me: Me; submitLabel: string; onSaved: (user: Me) => void | Promise<void> }) {
  const [form, setForm] = useState({
    display_name: me.display_name,
    birth_date: me.birth_date ?? "",
    gender: me.gender ?? "",
    prefecture: me.prefecture,
    bio: me.bio,
  });
  const [parts, setParts] = useState(me.parts);
  const [notify, setNotify] = useState(me.email_notifications);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);
  const set = (key: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    setForm({ ...form, [key]: e.target.value });
    setSaved(false);
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    setSaved(false);
    try {
      const res = await api<{ user: Me }>("/me", { method: "PATCH", body: { ...form, parts, email_notifications: notify } });
      await onSaved(res.user);
      setSaved(true);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <form onSubmit={submit} className="space-y-5">
      <Field label="表示名" required>
        <input className="field" value={form.display_name} onChange={set("display_name")} required maxLength={30} />
      </Field>
      <div className="grid gap-5 sm:grid-cols-2">
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
      <Field label="活動地域">
        <select className="field" value={form.prefecture} onChange={set("prefecture")}>
          <option value="">未設定</option>
          {PREFECTURES.map((p) => <option key={p}>{p}</option>)}
        </select>
      </Field>
      <div>
        <p className="mb-1.5 text-sm font-bold">担当パート</p>
        <ChipSelect label="担当パート" options={PARTS} value={parts} onChange={(v) => { setParts(v); setSaved(false); }} />
      </div>
      <Field label="自己紹介">
        <textarea className="field min-h-32" value={form.bio} onChange={set("bio")} maxLength={1000} placeholder="経験年数、使用機材、好きな音楽など" />
      </Field>
      <label className="flex cursor-pointer items-start gap-3 rounded-md border border-line bg-white p-3">
        <input type="checkbox" className="mt-0.5 h-5 w-5 accent-accent" checked={notify} onChange={(e) => { setNotify(e.target.checked); setSaved(false); }} />
        <span>
          <span className="block text-sm font-bold">メッセージが届いたらメールで通知する</span>
          <span className="block text-xs text-muted">通知先: {me.email}(未読がたまっても、スレッドごとに最初の1通だけ送ります)</span>
        </span>
      </label>
      <ErrorNote>{error}</ErrorNote>
      <div className="flex items-center gap-3">
        <Button type="submit" disabled={busy}>{submitLabel}</Button>
        {saved && <span role="status" className="text-sm font-bold text-join">保存しました</span>}
      </div>
    </form>
  );
}
