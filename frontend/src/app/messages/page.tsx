import type { Metadata } from "next";
import { ConversationList } from "./ConversationList";

export const metadata: Metadata = { title: "メッセージ" };

export default function Page() {
  return <ConversationList />;
}
