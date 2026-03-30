import { i18n } from "@/i18n"
import moment from "moment"

export function format_time(time: Date): string {
    time = moment(time).toDate()
    const locale = (i18n.global.locale as unknown as { value: string }).value || i18n.global.locale || 'ja'
    const formatted = new Intl.DateTimeFormat(locale, {
        year: 'numeric', month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit', second: '2-digit',
        hour12: false
    }).format(time)
    const day_of_week = [
        i18n.global.t("SUNDAY_TITLE"), i18n.global.t("MONDAY_TITLE"),
        i18n.global.t("TUESDAY_TITLE"), i18n.global.t("WEDNESDAY_TITLE"),
        i18n.global.t("THURSDAY_TITLE"), i18n.global.t("FRIDAY_TITLE"),
        i18n.global.t("SATURDAY_TITLE")
    ][time.getDay()]
    return `${formatted}(${day_of_week})`
}

export function format_number(n: number): string {
    const locale = (i18n.global.locale as unknown as { value: string }).value || i18n.global.locale || 'ja'
    return new Intl.NumberFormat(locale).format(n)
}

export function format_duration(duration_milli_second: number | null): string {
    if (!duration_milli_second || duration_milli_second === 0) {
        return ""
    }
    const total_seconds = Math.floor(duration_milli_second / 1000)
    const days = Math.floor(total_seconds / 86_400)
    const hours = Math.floor((total_seconds % 86_400) / 3_600)
    const minutes = Math.floor((total_seconds % 3_600) / 60)
    const seconds = total_seconds % 60
    const trimed_hours = Math.floor((total_seconds) / 3_600 * 100) / 100

    let diff_str = ""
    if (days !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += days + i18n.global.t("DAY_SUFFIX")
    }
    if (hours !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += hours + i18n.global.t("HOUR_SUFFIX")
    }
    if (minutes !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += minutes + i18n.global.t("MINUTE_SUFFIX")
    }
    if (diff_str === "") {
        diff_str += seconds + i18n.global.t("SECOND_SUFFIX")
    }
    if (diff_str !== "") {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += "<br>（" + trimed_hours + i18n.global.t("HOUR_SUFFIX") + "）"
    }
    return diff_str
}
