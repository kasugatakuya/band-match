import { Suspense } from "react";
import { Loading } from "@/components/ui";
import { UserProfile } from "./UserProfile";

export default function Page() {
  return (
    <Suspense fallback={<Loading />}>
      <UserProfile />
    </Suspense>
  );
}
