"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import useSWRInfinite from "swr/infinite";
import { fetcher, type Post } from "@/lib/api";
import { GENRES, PARTS, PREFECTURES } from "@/lib/constants";
import { PostCard } from "./PostCard";
import { Button, ErrorNote, Loading } from "./ui";

type Page = { posts: Post[]; next_cursor: string | null };

const FILTER_KEYS = ["kind", "prefecture", "part", "genre", "artist", "q"] as const;

const TABS = [
  { value: "", label: "すべて" },
  { value: "recruit", label: "メンバー募集" },
  { value: "join", label: "加入希望" },
];

/** 記事一覧と絞り込み。絞り込み条件は URL のクエリに持たせる(共有・戻るボタンで復元できる)。 */
export function PostList() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const filters = Object.fromEntries(FILTER_KEYS.map((k) => [k, searchParams.get(k) ?? ""])) as Record<(typeof FILTER_KEYS)[number], string>;
  const query = new URLSearchParams(Object.entries(filters).filter(([, v]) => v)).toString();
  const activeCount = FILTER_KEYS.filter((k) => k !== "kind" && filters[k]).length;

  const setFilters = (patch: Partial<typeof filters>) => {
    const next = new URLSearchParams(Object.entries({ ...filters, ...patch }).filter(([, v]) => v));
    const qs = next.toString();
    router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
  };

  const { data, error, isLoading, isValidating, size, setSize } = useSWRInfinite<Page>(
    (index, prev) => {
      if (prev && !prev.next_cursor) return null;
      const p = new URLSearchParams(query);
      if (prev?.next_cursor) p.set("cursor", prev.next_cursor);
      const qs = p.toString();
      return `/posts${qs ? `?${qs}` : ""}`;
    },
    fetcher,
    { revalidateFirstPage: false },
  );
  const posts = data?.flatMap((p) => p.posts) ?? [];
  const hasMore = !!data?.[data.length - 1]?.next_cursor;

  return (
    <div>
      <div role="tablist" aria-label="記事の種類" className="mb-4 flex gap-1 border-b border-line">
        {TABS.map((t) => (
          <button
            key={t.value}
            role="tab"
            aria-selected={filters.kind === t.value}
            onClick={() => setFilters({ kind: t.value })}
            className={`-mb-px border-b-[3px] px-3 py-2.5 text-sm font-bold transition-colors sm:px-4 ${
              filters.kind === t.value ? "border-accent text-ink" : "border-transparent text-muted hover:text-ink"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* key を変えて、URL が変わったら入力欄の中身も合わせ直す */}
      <form
        key={query}
        role="search"
        className="mb-6 grid grid-cols-2 gap-3 rounded-lg border border-line bg-card p-4 md:grid-cols-6"
        onSubmit={(e) => {
          e.preventDefault();
          const f = new FormData(e.currentTarget);
          setFilters({ artist: String(f.get("artist") ?? "").trim(), q: String(f.get("q") ?? "").trim() });
        }}
      >
        <label className="block md:col-span-2">
          <span className="mb-1 block text-xs font-bold">地域</span>
          <select className="field" value={filters.prefecture} onChange={(e) => setFilters({ prefecture: e.target.value })}>
            <option value="">すべての地域</option>
            {PREFECTURES.map((p) => <option key={p}>{p}</option>)}
          </select>
        </label>
        <label className="block md:col-span-2">
          <span className="mb-1 block text-xs font-bold">楽器パート</span>
          <select className="field" value={filters.part} onChange={(e) => setFilters({ part: e.target.value })}>
            <option value="">すべてのパート</option>
            {PARTS.map((p) => <option key={p}>{p}</option>)}
          </select>
        </label>
        <label className="col-span-2 block md:col-span-2">
          <span className="mb-1 block text-xs font-bold">ジャンル</span>
          <select className="field" value={filters.genre} onChange={(e) => setFilters({ genre: e.target.value })}>
            <option value="">すべてのジャンル</option>
            {GENRES.map((g) => <option key={g}>{g}</option>)}
          </select>
        </label>
        <label className="col-span-2 block md:col-span-2">
          <span className="mb-1 block text-xs font-bold">好きなアーティスト</span>
          <input name="artist" className="field" defaultValue={filters.artist} placeholder="例: ナンバーガール" maxLength={100} />
        </label>
        <label className="col-span-2 block md:col-span-3">
          <span className="mb-1 block text-xs font-bold">キーワード</span>
          <input name="q" className="field" defaultValue={filters.q} placeholder="タイトル・本文から探す" maxLength={100} />
        </label>
        <div className="col-span-2 flex items-end md:col-span-1">
          <Button type="submit" variant="dark" className="w-full">検索</Button>
        </div>
      </form>

      {activeCount > 0 && (
        <p className="mb-4 text-sm">
          <button type="button" className="font-bold underline" onClick={() => setFilters({ prefecture: "", part: "", genre: "", artist: "", q: "" })}>
            絞り込みを解除
          </button>
        </p>
      )}

      {error ? (
        <ErrorNote>記事を読み込めませんでした。時間をおいて再読み込みしてください。</ErrorNote>
      ) : isLoading ? (
        <Loading />
      ) : posts.length === 0 ? (
        <div className="rounded-lg border border-dashed border-line px-4 py-14 text-center">
          <p className="font-bold">条件に合う記事はまだありません</p>
          <p className="mt-1 text-sm text-muted">{activeCount > 0 ? "条件をゆるめて探してみてください。" : "最初の記事を書いてみませんか。"}</p>
        </div>
      ) : (
        <>
          <ul className="grid gap-4 md:grid-cols-2">
            {posts.map((post) => (
              <li key={post.id}><PostCard post={post} /></li>
            ))}
          </ul>
          {hasMore && (
            <div className="mt-6 text-center">
              <Button variant="outline" disabled={isValidating} onClick={() => setSize(size + 1)}>
                {isValidating ? "読み込み中…" : "もっと見る"}
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
