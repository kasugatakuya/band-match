import type { Metadata } from "next";
import { Onboarding } from "./Onboarding";

export const metadata: Metadata = { title: "プロフィールの入力" };

export default function Page() {
  return <Onboarding />;
}
