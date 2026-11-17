import type { Metadata } from "next";
import { Suspense } from "react";
import { Loading } from "@/components/ui";
import { EditPost } from "./EditPost";

export const metadata: Metadata = { title: "記事を編集" };

export default function Page() {
  return (
    <Suspense fallback={<Loading />}>
      <EditPost />
    </Suspense>
  );
}
