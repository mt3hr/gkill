import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { GkillAPI } from "../../classes/api/gkill-api";
import type { ApplicationConfig } from "../../classes/datas/config/application-config";
import type { Kyou } from "@/classes/datas/kyou";

export default interface RyuuListItemViewProps {
    gkill_api: GkillAPI
    application_config: ApplicationConfig
    find_kyou_query_default: FindKyouQuery
    target_kyou: Kyou | null
    enable_context_menu: boolean
    enable_dialog: boolean
    abort_controller: AbortController
    editable: boolean
}