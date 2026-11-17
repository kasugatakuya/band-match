"use client";

import { useState } from "react";

/** 複数選択できるボタンの並び(パート・ジャンル用)。 */
export function ChipSelect({ options, value, onChange, label }: { options: readonly string[]; value: string[]; onChange: (v: string[]) => void; label: string }) {
  return (
    <div role="group" aria-label={label} className="flex flex-wrap gap-2">
      {options.map((opt) => {
        const on = value.includes(opt);
        return (
          <button
            key={opt}
            type="button"
            aria-pressed={on}
            onClick={() => onChange(on ? value.filter((v) => v !== opt) : [...value, opt])}
            className={`rounded-full border px-3 py-2 text-sm leading-none transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink ${
              on ? "border-ink bg-ink font-bold text-white" : "border-line bg-white text-ink hover:border-ink/50"
            }`}
          >
            {opt}
          </button>
        );
      })}
    </div>
  );
}

/** 自由入力のタグ(好きなアーティスト用)。Enter か「、」「,」で追加。 */
export function TagInput({ value, onChange, placeholder, max = 20 }: { value: string[]; onChange: (v: string[]) => void; placeholder?: string; max?: number }) {
  const [draft, setDraft] = useState("");

  const add = (raw: string) => {
    const names = raw.split(/[,、]/).map((s) => s.trim().slice(0, 50)).filter(Boolean);
    const next = [...value];
    for (const name of names) {
      if (next.length < max && !next.some((v) => v.toLowerCase() === name.toLowerCase())) next.push(name);
    }
    onChange(next);
    setDraft("");
  };

  return (
    <div>
      {value.length > 0 && (
        <ul className="mb-2 flex flex-wrap gap-2">
          {value.map((tag) => (
            <li key={tag} className="flex items-center gap-1 rounded-full bg-ink py-1 pl-3 pr-1 text-sm text-white">
              {tag}
              <button
                type="button"
                aria-label={`${tag} を削除`}
                onClick={() => onChange(value.filter((v) => v !== tag))}
                className="flex h-6 w-6 items-center justify-center rounded-full hover:bg-white/20"
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}
      <input
        className="field"
        value={draft}
        placeholder={placeholder}
        disabled={value.length >= max}
        onChange={(e) => (/[,、]/.test(e.target.value) ? add(e.target.value) : setDraft(e.target.value))}
        onKeyDown={(e) => {
          // 日本語入力の変換確定の Enter では追加しない
          if (e.key === "Enter" && !e.nativeEvent.isComposing) {
            e.preventDefault();
            add(draft);
          }
        }}
        onBlur={() => draft.trim() && add(draft)}
      />
    </div>
  );
}
