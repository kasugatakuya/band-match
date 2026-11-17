import type { Metadata } from "next";
import { Zen_Kaku_Gothic_New } from "next/font/google";
import Link from "next/link";
import { Header, VerifyBanner } from "@/components/Header";
import "./globals.css";

const zen = Zen_Kaku_Gothic_New({
  variable: "--font-zen",
  weight: ["400", "500", "700", "900"],
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: { default: "BandMatch | バンドメンバー募集・加入希望", template: "%s | BandMatch" },
  description: "バンドメンバーを探している人と、バンドに入りたい人をつなぐ募集サイト。地域・パート・ジャンル・好きなアーティストで探せます。",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="ja" className={`${zen.variable} h-full antialiased`}>
      <body className="flex min-h-full flex-col font-sans">
        <Header />
        <VerifyBanner />
        <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-8">{children}</main>
        <footer className="border-t border-line">
          <div className="mx-auto flex max-w-5xl flex-wrap items-center gap-x-6 gap-y-2 px-4 py-6 text-xs text-muted">
            <span>© BandMatch</span>
            <Link href="/" className="hover:text-ink">記事を探す</Link>
            <Link href="/posts/new" className="hover:text-ink">記事を書く</Link>
          </div>
        </footer>
      </body>
    </html>
  );
}
