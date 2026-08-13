import type { GkillAPI } from "@/classes/api/gkill-api"
import type { ApplicationConfig } from "@/classes/datas/config/application-config"
import type { DnoteCorrelationGraphQuery } from "@/classes/dnote/dnote-correlation"

export default interface DnoteCorrelationGraphEditorViewProps {
    gkill_api: GkillAPI
    application_config: ApplicationConfig
    initial_query: DnoteCorrelationGraphQuery
}
