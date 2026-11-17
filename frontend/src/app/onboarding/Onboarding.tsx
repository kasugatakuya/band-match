"use client";

import { useRouter } from "next/navigation";
import { ProfileForm } from "@/components/ProfileForm";
import { Loading, PageTitle } from "@/components/ui";
import { useRequireAuth } from "@/lib/auth";

/** Google で登録した直後など、生年月日・性別が未入力のときに通る画面。 */
export function Onboarding() {
  const router = useRouter();
  const { me, mutate } = useRequireAuth({ allowIncompleteProfile: true });
  if (!me) return <Loading />;
  return (
    <div className="mx-auto max-w-2xl">
      <PageTitle sub="記事の投稿やメッセージの送信には、生年月日と性別の登録が必要です。">プロフィールを入力してください</PageTitle>
      <ProfileForm
        me={me}
        submitLabel="登録してはじめる"
        onSaved={async (user) => {
          await mutate({ user }, { revalidate: false });
          router.replace("/");
        }}
      />
    </div>
  );
}
