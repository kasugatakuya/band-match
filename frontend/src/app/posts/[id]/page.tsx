import { Suspense } from "react";
import { Loading } from "@/components/ui";
import { PostDetail } from "./PostDetail";

export default function Page() {
  return (
    <Suspense fallback={<Loading />}>
      <PostDetail />
    </Suspense>
  );
}
