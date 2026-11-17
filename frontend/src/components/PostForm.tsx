"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { useSWRConfig } from "swr";
import { api, errorMessage, type Post } from "@/lib/api";
import { GENRES, PARTS, PREFECTURES, type PostKind } from "@/lib/constants";
import { ChipSelect, TagInput } from "./inputs";
import { Button, ErrorNote, Field } from "./ui";

const KINDS: { value: PostKind; label: string; desc: string }[] = [
  { value: "recruit", label: "メンバー募集", desc: "バンドのメンバーを探している" },
  { value: "join", label: "加入希望", desc: "入れるバンドを探している" },
];

/** 記事の新規作成・編集フォーム。post を渡すと編集になる。 */
export function PostForm({ post, defaultPrefecture = "" }: { post?: Post; defaultPrefecture?: string }) {
  const router = useRouter();
  const { mutate } = useSWRConfig();
  const [kind, setKind] = useState<PostKind>(post?.kind ?? "recruit");
  const [title, setTitle] = useState(post?.title ?? "");
  const [body, setBody] = useState(post?.body ?? "");
  const [prefecture, setPrefecture] = useState(post?.prefecture ?? defaultPrefecture);
  const [parts, setParts] = useState<string[]>(post?.parts ?? []);
  const [genres, setGenres] = useState<string[]>(post?.genres ?? []);
  const [artists, setArtists] = useState<string[]>(post?.artists ?? []);
  const [status, setStatus] = useState(post?.status ?? "open");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  // 一覧や詳細に古い内容が残らないよう、記事関連のキャッシュを捨てる
  const refreshPosts = () => mutate((key) => typeof key === "string" && (key.startsWith("/posts") || key.startsWith("/me/posts") || key.includes("/posts")), undefined, { revalidate: true });

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (parts.length === 0) {
      setError("パートを1つ以上選択してください。");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const payload = { kind, title, body, prefecture, parts, genres, artists, status };
      const res = post
        ? await api<{ post: Post }>(`/posts/${post.id}`, { method: "PATCH", body: payload })
        : await api<{ post: Post }>("/posts", { method: "POST", body: payload });
      await refreshPosts();
      router.push(`/posts/${res.post.id}`);
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  const remove = async () => {
    if (!post || !window.confirm("この記事を削除します。元に戻せませんが、よろしいですか?\n(この記事から始まったメッセージのやり取りは残ります)")) return;
    setBusy(true);
    setError("");
    try {
      await api(`/posts/${post.id}`, { method: "DELETE" });
      await refreshPosts();
      router.push("/mypage");
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <form onSubmit={submit} className="space-y-6">
      <fieldset>
        <legend className="mb-1.5 text-sm font-bold">記事の種類</legend>
        <div className="grid gap-3 sm:grid-cols-2">
          {KINDS.map((k) => (
            <label
              key={k.value}
              className={`cursor-pointer rounded-lg border-2 bg-white p-4 transition-colors has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-ink ${
                kind === k.value ? (k.value === "recruit" ? "border-accent" : "border-join") : "border-line hover:border-ink/40"
              }`}
            >
              <input type="radio" name="kind" className="sr-only" checked={kind === k.value} onChange={() => setKind(k.value)} />
              <span className="block font-black">{k.label}</span>
              <span className="mt-0.5 block text-sm text-muted">{k.desc}</span>
            </label>
          ))}
        </div>
        <p className="mt-1.5 text-xs text-muted">メンバー募集・加入希望は、それぞれ3件まで投稿できます(締め切った記事も数に含みます)。</p>
      </fieldset>

      <Field label="タイトル" required>
        <input
          className="field"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          required
          maxLength={80}
          placeholder={kind === "recruit" ? "例: オルタナ系バンドでドラムを募集しています" : "例: 都内で活動できるバンドを探しているベースです"}
        />
      </Field>

      <Field label="地域" required>
        <select className="field" value={prefecture} onChange={(e) => setPrefecture(e.target.value)} required>
          <option value="" disabled>選択してください</option>
          {PREFECTURES.map((p) => <option key={p}>{p}</option>)}
        </select>
      </Field>

      <div>
        <p className="mb-1.5 flex items-baseline gap-2 text-sm font-bold">
          {kind === "recruit" ? "募集するパート" : "自分のパート"}
          <span className="rounded bg-accent px-1.5 py-0.5 text-[11px] leading-none text-white">必須</span>
        </p>
        <ChipSelect label={kind === "recruit" ? "募集するパート" : "自分のパート"} options={PARTS} value={parts} onChange={setParts} />
      </div>

      <div>
        <p className="mb-1.5 text-sm font-bold">ジャンル</p>
        <ChipSelect label="ジャンル" options={GENRES} value={genres} onChange={setGenres} />
      </div>

      <div>
        <p className="mb-1.5 text-sm font-bold">好きなアーティスト</p>
        <TagInput value={artists} onChange={setArtists} placeholder="入力して Enter で追加(20件まで)" />
        <p className="mt-1.5 text-xs text-muted">検索で見つけてもらいやすくなります。</p>
      </div>

      <Field label="本文" required hint="活動頻度、練習場所、目標(ライブ・音源制作など)、経験年数などを書くと伝わりやすくなります。連絡先はメッセージでやり取りできるので、書かなくて大丈夫です。">
        <textarea className="field min-h-56" value={body} onChange={(e) => setBody(e.target.value)} required maxLength={4000} />
      </Field>

      {post && (
        <Field label="募集状況" hint="締め切ると一覧に表示されなくなり、新しいメッセージも受け付けません。">
          <select className="field" value={status} onChange={(e) => setStatus(e.target.value as "open" | "closed")}>
            <option value="open">募集中</option>
            <option value="closed">締め切り</option>
          </select>
        </Field>
      )}

      <ErrorNote>{error}</ErrorNote>

      <div className="flex flex-wrap items-center gap-3">
        <Button type="submit" disabled={busy}>{post ? "変更を保存" : "この内容で投稿"}</Button>
        <Button type="button" variant="ghost" onClick={() => router.back()}>キャンセル</Button>
        {post && <Button type="button" variant="danger" className="ml-auto" disabled={busy} onClick={remove}>記事を削除</Button>}
      </div>
    </form>
  );
}
