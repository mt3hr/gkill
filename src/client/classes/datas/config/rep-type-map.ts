import { i18n } from "@/i18n"

export function generate_rep_type_map(): Map<string, string> {
    const map = new Map<string, string>()
    map.set("Files", i18n.global.t("REP_TYPE_FILES"))
    map.set("Kmemo", i18n.global.t("REP_TYPE_KMEMO"))
    map.set("KC", i18n.global.t("REP_TYPE_KC"))
    map.set("URLog", i18n.global.t("REP_TYPE_URLOG"))
    map.set("Nlog", i18n.global.t("REP_TYPE_NLOG"))
    map.set("TimeIs", i18n.global.t("REP_TYPE_TIMEIS"))
    map.set("Mi", i18n.global.t("REP_TYPE_MI"))
    map.set("Lantana", i18n.global.t("REP_TYPE_LANTANA"))
    map.set("ReKyou", i18n.global.t("REP_TYPE_REKYOU"))
    map.set("MiReKyou", i18n.global.t("REP_TYPE_MIREKYOU"))
    return map
}
