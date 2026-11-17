import type { Metadata } from "next";
import { MyPage } from "./MyPage";

export const metadata: Metadata = { title: "マイページ" };

export default function Page() {
  return <MyPage />;
}
