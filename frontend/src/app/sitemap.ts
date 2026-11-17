import type { MetadataRoute } from "next";
import { cacheLife } from "next/cache";
import { siteUrl } from "@/lib/site";

const backendUrl = (process.env.BACKEND_URL ?? "http://localhost:8080").replace(/\/$/, "");

// サイトマップ1ファイルの上限(5万件)より十分少なく抑える
const MAX_POSTS = 5000;

type PostEntry = { id: string; updated_at: string };

// 募集中の記事を新しい順に集める。結果は1時間キャッシュする。
async function listOpenPosts(): Promise<PostEntry[]> {
  "use cache";
  cacheLife("hours");

  const posts: PostEntry[] = [];
  let cursor: string | null = null;
  while (posts.length < MAX_POSTS) {
    const params = new URLSearchParams({ limit: "50" });
    if (cursor) params.set("cursor", cursor);
    const res = await fetch(`${backendUrl}/api/posts?${params}`);
    if (!res.ok) throw new Error(`記事一覧の取得に失敗しました: ${res.status}`);
    const data: { posts: PostEntry[]; next_cursor: string | null } = await res.json();
    for (const p of data.posts) posts.push({ id: p.id, updated_at: p.updated_at });
    if (!data.next_cursor) break;
    cursor = data.next_cursor;
  }
  return posts.slice(0, MAX_POSTS);
}

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const base = siteUrl();

  // API に届かないときでも、固定ページだけのサイトマップを返す
  let posts: PostEntry[] = [];
  try {
    posts = await listOpenPosts();
  } catch (err) {
    console.error(err);
  }

  return [
    { url: `${base}/`, changeFrequency: "hourly", priority: 1 },
    { url: `${base}/signup`, changeFrequency: "yearly", priority: 0.3 },
    { url: `${base}/login`, changeFrequency: "yearly", priority: 0.3 },
    ...posts.map((p) => ({
      url: `${base}/posts/${p.id}`,
      lastModified: p.updated_at,
      changeFrequency: "daily" as const,
      priority: 0.7,
    })),
  ];
}
