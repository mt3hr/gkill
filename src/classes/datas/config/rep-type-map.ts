import { i18n } from "@/i18n"

export async function generate_rep_type_map(): Promise<Map<string, string>> {
    const map = new Map<string, string>()
    map.set("Kmemo", i18n.global.t("KFTL_APP_NAME"))
    map.set("URLog", i18n.global.t("URLOG_APP_NAME"))
    map.set("Nlog", i18n.global.t("NLOG_APP_NAME"))
    map.set("TimeIs", i18n.global.t("TIMEIS_APP_NAME"))
    map.set("Mi", i18n.global.t("MI_APP_NAME"))
    map.set("Lantana", i18n.global.t("LANTANA_APP_NAME"))
    map.set("ReKyou", i18n.global.t("REKYOU_TITLE"))
    return map
}