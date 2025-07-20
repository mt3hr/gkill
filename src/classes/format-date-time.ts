import { i18n } from "@/i18n"
import moment from "moment"

export function format_time(time: Date): string {
    time = moment(time).toDate()
    const year: string | number = time.getFullYear()
    let month: string | number = time.getMonth() + 1
    let date: string | number = time.getDate()
    let hour: string | number = time.getHours()
    let minute: string | number = time.getMinutes()
    let second: string | number = time.getSeconds()
    const day_of_week = [i18n.global.t("SUNDAY_TITLE"), i18n.global.t("MONDAY_TITLE"), i18n.global.t("TUESDAY_TITLE"), i18n.global.t("WEDNESDAY_TITLE"), i18n.global.t("THURSDAY_TITLE"), i18n.global.t("FRIDAY_TITLE"), i18n.global.t("SATURDAY_TITLE")][time.getDay()]
    month = ('0' + month).slice(-2)
    date = ('0' + date).slice(-2)
    hour = ('0' + hour).slice(-2)
    minute = ('0' + minute).slice(-2)
    second = ('0' + second).slice(-2)
    return year + '/' + month + '/' + date + '(' + day_of_week + ')' + ' ' + hour + ':' + minute + ':' + second
}

export function format_duration(duration_milli_second: any | null): string {
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
