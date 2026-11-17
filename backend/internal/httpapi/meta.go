package httpapi

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// 選択肢。frontend/src/lib/constants.ts と同じ内容にしておくこと。
var prefectures = []string{
	"北海道", "青森県", "岩手県", "宮城県", "秋田県", "山形県", "福島県",
	"茨城県", "栃木県", "群馬県", "埼玉県", "千葉県", "東京都", "神奈川県",
	"新潟県", "富山県", "石川県", "福井県", "山梨県", "長野県", "岐阜県", "静岡県", "愛知県",
	"三重県", "滋賀県", "京都府", "大阪府", "兵庫県", "奈良県", "和歌山県",
	"鳥取県", "島根県", "岡山県", "広島県", "山口県",
	"徳島県", "香川県", "愛媛県", "高知県",
	"福岡県", "佐賀県", "長崎県", "熊本県", "大分県", "宮崎県", "鹿児島県", "沖縄県",
}

var partOptions = []string{
	"ボーカル", "ギター", "ベース", "ドラム", "キーボード", "DJ・マニピュレーター",
	"管楽器", "弦楽器", "パーカッション", "その他",
}

var genreOptions = []string{
	"ロック", "ポップス", "パンク", "メタル", "ハードコア", "オルタナティブ", "エモ",
	"ジャズ", "フュージョン", "ファンク", "ソウル・R&B", "ブルース", "ヒップホップ",
	"レゲエ・スカ", "フォーク・アコースティック", "ヴィジュアル系", "アニソン・ボカロ",
	"エレクトロ", "インスト", "コピーバンド", "その他",
}

var genderOptions = []string{"male", "female", "other", "no_answer"}

var jst = time.FixedZone("JST", 9*60*60)

const (
	minAge = 10
	maxAge = 100
)

// ageOn は誕生日から満年齢を求める(日本時間基準)。
func ageOn(birth time.Time, now time.Time) int {
	now = now.In(jst)
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}

func agePtr(birth *time.Time) *int {
	if birth == nil {
		return nil
	}
	a := ageOn(*birth, time.Now())
	return &a
}

// parseBirthDate は YYYY-MM-DD を検証して返す。
func parseBirthDate(s string) (time.Time, string) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return t, "生年月日を正しく入力してください。"
	}
	if age := ageOn(t, time.Now()); age < minAge || age > maxAge {
		return t, "生年月日を正しく入力してください。"
	}
	return t, ""
}

func cleanText(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\x00", ""))
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

// cleanChoices は選択式の項目を検証する。重複は取り除く。
func cleanChoices(in, allowed []string, max int) ([]string, bool) {
	out := []string{}
	for _, v := range in {
		if !slices.Contains(allowed, v) {
			return nil, false
		}
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out, len(out) <= max
}

// cleanTags は自由入力のタグ(好きなアーティスト)を整える。
func cleanTags(in []string, max, maxLen int) ([]string, bool) {
	out := []string{}
	for _, v := range in {
		v = cleanText(v)
		if v == "" {
			continue
		}
		if runeLen(v) > maxLen {
			return nil, false
		}
		if !slices.ContainsFunc(out, func(o string) bool { return strings.EqualFold(o, v) }) {
			out = append(out, v)
		}
	}
	return out, len(out) <= max
}

// likeEscape は ILIKE のワイルドカードを無効化する。
func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
