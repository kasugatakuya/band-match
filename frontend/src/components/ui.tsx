import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";

const buttonBase =
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md px-4 py-2.5 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink";

const variants = {
  primary: "bg-accent text-white hover:bg-accent-dark",
  dark: "bg-ink text-white hover:bg-ink/85",
  outline: "border border-ink/25 bg-white text-ink hover:border-ink",
  ghost: "text-ink hover:bg-ink/5",
  danger: "border border-accent/40 text-accent hover:bg-accent/5",
};

type Variant = keyof typeof variants;

export function Button({ variant = "primary", className = "", ...props }: ComponentProps<"button"> & { variant?: Variant }) {
  return <button {...props} className={`${buttonBase} ${variants[variant]} ${className}`} />;
}

export function ButtonLink({ variant = "primary", className = "", ...props }: ComponentProps<typeof Link> & { variant?: Variant }) {
  return <Link {...props} className={`${buttonBase} ${variants[variant]} ${className}`} />;
}

export function Field({ label, hint, required, children }: { label: string; hint?: string; required?: boolean; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1.5 flex items-baseline gap-2 text-sm font-bold">
        {label}
        {required && <span className="rounded bg-accent px-1.5 py-0.5 text-[11px] leading-none text-white">必須</span>}
      </span>
      {children}
      {hint && <span className="mt-1.5 block text-xs text-muted">{hint}</span>}
    </label>
  );
}

export function ErrorNote({ children }: { children: ReactNode }) {
  if (!children) return null;
  return (
    <p role="alert" className="rounded-md border border-accent/30 bg-accent/5 px-3 py-2.5 text-sm text-accent-dark">
      {children}
    </p>
  );
}

export function Notice({ children }: { children: ReactNode }) {
  return <p className="rounded-md border border-line bg-white px-3 py-2.5 text-sm text-ink">{children}</p>;
}

export function Loading({ label = "読み込み中…" }: { label?: string }) {
  return <p className="py-16 text-center text-sm text-muted">{label}</p>;
}

export function Avatar({ name, url, size = 40 }: { name: string; url?: string; size?: number }) {
  const style = { width: size, height: size, fontSize: size * 0.42 };
  if (url) {
    // Google のプロフィール画像など外部 URL をそのまま表示する
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={url} alt="" referrerPolicy="no-referrer" style={style} className="shrink-0 rounded-full bg-line object-cover" />;
  }
  return (
    <span aria-hidden style={style} className="flex shrink-0 items-center justify-center rounded-full bg-ink font-bold text-paper">
      {[...name][0] ?? "?"}
    </span>
  );
}

export function KindBadge({ kind }: { kind: "recruit" | "join" }) {
  return (
    <span className={`inline-block rounded-sm px-2 py-1 text-xs font-black leading-none text-white ${kind === "recruit" ? "bg-accent" : "bg-join"}`}>
      {kind === "recruit" ? "メンバー募集" : "加入希望"}
    </span>
  );
}

export function Chip({ children }: { children: ReactNode }) {
  return <span className="inline-block rounded-full border border-line bg-white px-2.5 py-1 text-xs leading-none text-ink">{children}</span>;
}

export function PageTitle({ children, sub }: { children: ReactNode; sub?: ReactNode }) {
  return (
    <div className="mb-6">
      <h1 className="text-2xl font-black tracking-tight sm:text-3xl">{children}</h1>
      {sub && <p className="mt-2 text-sm text-muted">{sub}</p>}
    </div>
  );
}
