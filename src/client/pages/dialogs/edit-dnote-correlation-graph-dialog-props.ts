import type { GkillAPI } from "../../classes/api/gkill-api"
import type { ApplicationConfig } from "../../classes/datas/config/application-config"
import type { DnoteCorrelationGraphQuery } from "../../classes/dnote/dnote-correlation"

export default interface EditDnoteCorrelationGraphDialogProps {
    gkill_api: GkillAPI
    application_config: ApplicationConfig
    dnote_correlation_graph_query: DnoteCorrelationGraphQuery
}
