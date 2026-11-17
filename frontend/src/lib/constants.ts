// 選択肢。backend/internal/httpapi/meta.go と同じ内容にしておくこと。

export const PREFECTURES = [
  "北海道", "青森県", "岩手県", "宮城県", "秋田県", "山形県", "福島県",
  "茨城県", "栃木県", "群馬県", "埼玉県", "千葉県", "東京都", "神奈川県",
  "新潟県", "富山県", "石川県", "福井県", "山梨県", "長野県", "岐阜県", "静岡県", "愛知県",
  "三重県", "滋賀県", "京都府", "大阪府", "兵庫県", "奈良県", "和歌山県",
  "鳥取県", "島根県", "岡山県", "広島県", "山口県",
  "徳島県", "香川県", "愛媛県", "高知県",
  "福岡県", "佐賀県", "長崎県", "熊本県", "大分県", "宮崎県", "鹿児島県", "沖縄県",
] as const;

export const PARTS = [
  "ボーカル", "ギター", "ベース", "ドラム", "キーボード", "DJ・マニピュレーター",
  "管楽器", "弦楽器", "パーカッション", "その他",
] as const;

export const GENRES = [
  "ロック", "ポップス", "パンク", "メタル", "ハードコア", "オルタナティブ", "エモ",
  "ジャズ", "フュージョン", "ファンク", "ソウル・R&B", "ブルース", "ヒップホップ",
  "レゲエ・スカ", "フォーク・アコースティック", "ヴィジュアル系", "アニソン・ボカロ",
  "エレクトロ", "インスト", "コピーバンド", "その他",
] as const;

export const GENDERS = [
  { value: "male", label: "男性" },
  { value: "female", label: "女性" },
  { value: "other", label: "その他" },
  { value: "no_answer", label: "回答しない" },
] as const;

export type PostKind = "recruit" | "join";

export const KIND_LABEL: Record<PostKind, string> = {
  recruit: "メンバー募集",
  join: "加入希望",
};

export function genderLabel(value: string | null | undefined): string {
  return GENDERS.find((g) => g.value === value)?.label ?? "";
}
