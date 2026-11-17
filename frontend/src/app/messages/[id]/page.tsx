import type { Metadata } from "next";
import { Suspense } from "react";
import { Loading } from "@/components/ui";
import { Chat } from "./Chat";

export const metadata: Metadata = { title: "メッセージ" };

export default function Page() {
  return (
    <Suspense fallback={<Loading />}>
      <Chat />
    </Suspense>
  );
}
