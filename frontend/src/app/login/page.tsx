import type { Metadata } from "next";
import { Suspense } from "react";
import { Loading } from "@/components/ui";
import { LoginForm } from "./LoginForm";

export const metadata: Metadata = { title: "ログイン" };

export default function Page() {
  return (
    <Suspense fallback={<Loading />}>
      <LoginForm />
    </Suspense>
  );
}
