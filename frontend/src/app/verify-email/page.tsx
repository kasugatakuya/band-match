import type { Metadata } from "next";
import { Suspense } from "react";
import { Loading } from "@/components/ui";
import { VerifyEmail } from "./VerifyEmail";

export const metadata: Metadata = { title: "メールアドレスの確認" };

export default function Page() {
  return (
    <Suspense fallback={<Loading />}>
      <VerifyEmail />
    </Suspense>
  );
}
