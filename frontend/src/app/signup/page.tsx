import type { Metadata } from "next";
import { SignupForm } from "./SignupForm";

export const metadata: Metadata = { title: "新規登録" };

export default function Page() {
  return <SignupForm />;
}
