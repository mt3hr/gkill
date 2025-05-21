import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { GkillAPI } from "../../classes/api/gkill-api";
import type { ApplicationConfig } from "../../classes/datas/config/application-config";

export default interface RyuuListViewProps {
    gkill_api: GkillAPI
    application_config: ApplicationConfig
    find_kyou_query_default: FindKyouQuery
    related_time: Date | null
    editable: boolean
}