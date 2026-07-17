import type { GkillAPI } from "../../classes/api/gkill-api"
import type { ApplicationConfig } from "../../classes/datas/config/application-config"
import type DnoteTrendGraphQuery from "../views/dnote-trend-graph-query"

export default interface EditDnoteTrendGraphDialogProps {
    gkill_api: GkillAPI
    application_config: ApplicationConfig
    dnote_trend_graph_query: DnoteTrendGraphQuery
}
