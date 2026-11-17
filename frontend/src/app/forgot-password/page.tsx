import type { Metadata } from "next";
import { ForgotForm } from "./ForgotForm";

export const metadata: Metadata = { title: "パスワードの再設定" };

export default function Page() {
  return <ForgotForm />;
}
