import useSWR from "swr";
import type { PostKind } from "./constants";

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}

/** Go の API を呼ぶ。失敗したら ApiError(日本語メッセージつき)を投げる。 */
export async function api<T>(path: string, init?: { method?: string; body?: unknown }): Promise<T> {
  const hasBody = init?.body !== undefined;
  let res: Response;
  try {
    res = await fetch(`/api${path}`, {
      method: init?.method ?? "GET",
      headers: hasBody ? { "Content-Type": "application/json" } : undefined,
      body: hasBody ? JSON.stringify(init.body) : undefined,
      credentials: "same-origin",
    });
  } catch {
    throw new ApiError(0, "network", "通信に失敗しました。接続を確認してもう一度お試しください。");
  }
  const data = (await res.json().catch(() => null)) as { error?: string; code?: string } | null;
  if (!res.ok) {
    throw new ApiError(res.status, data?.code ?? "unknown", data?.error ?? "エラーが発生しました。");
  }
  return data as T;
}

export const fetcher = <T,>(path: string) => api<T>(path);

export function errorMessage(e: unknown): string {
  return e instanceof ApiError ? e.message : "エラーが発生しました。";
}

export type Me = {
  id: string;
  email: string;
  email_verified: boolean;
  has_password: boolean;
  google_linked: boolean;
  display_name: string;
  avatar_url: string;
  bio: string;
  prefecture: string;
  parts: string[];
  birth_date: string | null;
  age: number | null;
  gender: string | null;
  email_notifications: boolean;
  profile_complete: boolean;
};

export type PublicUser = {
  id: string;
  display_name: string;
  avatar_url: string;
  bio: string;
  prefecture: string;
  parts: string[];
  age: number | null;
  gender: string | null;
  created_at: string;
};

export type Author = Pick<PublicUser, "id" | "display_name" | "avatar_url" | "age" | "gender">;

export type Post = {
  id: string;
  kind: PostKind;
  title: string;
  body: string;
  prefecture: string;
  parts: string[];
  genres: string[];
  artists: string[];
  status: "open" | "closed";
  created_at: string;
  updated_at: string;
  author: Author;
};

export type Conversation = {
  id: string;
  post_id: string | null;
  post_title: string;
  post_kind: PostKind | "";
  other: { id: string; display_name: string; avatar_url: string };
  last_message_at?: string;
  last_body?: string;
  unread?: number;
};

export type Message = { id: number; sender_id: string; body: string; created_at: string };

/** ログイン中のユーザー。未ログインなら me は null。 */
export function useMe() {
  const { data, isLoading, mutate } = useSWR<{ user: Me | null }>("/me", fetcher);
  return { me: data?.user ?? null, loading: isLoading, mutate };
}
