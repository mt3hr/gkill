import type { Kyou } from "@/classes/datas/kyou";
import { i18n } from "@/i18n";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class RelatedWeekGetter implements DnoteKeyGetter {

    static from_json(_json: Record<string, unknown>): RelatedWeekGetter {
        return new RelatedWeekGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        // 週番号だけだと年が落ちるため、去年の第33週と今年の第33週が同じキーになって合算されていた。
        // ISO週の月曜〜日曜の日付範囲を見出しにして、年が違えば別グループになるようにする
        const monday = this.get_start_of_iso_week(loaded_kyou.related_time)
        const sunday = new Date(monday.getFullYear(), monday.getMonth(), monday.getDate() + 6)
        const start = `${monday.getFullYear()}/${(monday.getMonth() + 1).toString().padStart(2, '0')}/${monday.getDate().toString().padStart(2, '0')}`
        const end = `${(sunday.getMonth() + 1).toString().padStart(2, '0')}/${sunday.getDate().toString().padStart(2, '0')}`
        return [i18n.global.t("DNOTE_RELATED_WEEK_RANGE_TITLE", { start, end })]
    }

    to_json() {
        return {
            type: "RelatedWeekGetter",
        }
    }

    // ISO週（月曜始まり）の月曜0時を返す。
    // トレンドグラフの週粒度 moment(...).startOf('isoWeek') と同じ区切り。
    // 日付コンポーネント演算で求めるのでDSTのある地域でもずれない
    get_start_of_iso_week(date: Date): Date {
        const offset_from_monday = (date.getDay() + 6) % 7
        return new Date(date.getFullYear(), date.getMonth(), date.getDate() - offset_from_monday)
    }

}
