import type { Metadata } from "next";
import { Suspense } from "react";
import { Loading } from "@/components/ui";
import { ResetForm } from "./ResetForm";

export const metadata: Metadata = { title: "新しいパスワードの設定" };

export default function Page() {
  return (
    <Suspense fallback={<Loading />}>
      <ResetForm />
    </Suspense>
  );
}
