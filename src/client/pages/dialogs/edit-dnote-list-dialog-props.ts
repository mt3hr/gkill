import type { GkillAPI } from "../../classes/api/gkill-api"
import type { ApplicationConfig } from "../../classes/datas/config/application-config"
import type DnoteListQuery from "../views/dnote-list-query"

export default interface EditDnoteListDialogProps {
    gkill_api: GkillAPI
    application_config: ApplicationConfig
    dnote_list_query: DnoteListQuery
}