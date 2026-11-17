import type { MetadataRoute } from "next";
import { siteUrl } from "@/lib/site";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      // ログインが必要な画面と、メール内リンク用の画面は検索対象にしない
      disallow: ["/api/", "/messages", "/mypage", "/onboarding", "/posts/new", "/verify-email", "/reset-password", "/forgot-password"],
    },
    sitemap: `${siteUrl()}/sitemap.xml`,
  };
}
