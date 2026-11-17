// サイトの公開 URL(末尾スラッシュなし)。sitemap.xml と robots.txt に使う。
// SITE_URL があればそれを使い、なければ Vercel が自動で入れる本番ドメインを使う。
export function siteUrl(): string {
  const explicit = process.env.SITE_URL?.replace(/\/$/, "");
  if (explicit) return explicit;
  const vercel = process.env.VERCEL_PROJECT_PRODUCTION_URL;
  if (vercel) return `https://${vercel}`;
  return "http://localhost:3000";
}
