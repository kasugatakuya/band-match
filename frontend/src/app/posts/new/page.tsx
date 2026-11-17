import type { Metadata } from "next";
import { NewPost } from "./NewPost";

export const metadata: Metadata = { title: "記事を書く" };

export default function Page() {
  return <NewPost />;
}
