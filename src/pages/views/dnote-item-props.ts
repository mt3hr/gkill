import type { GkillAPI } from "../../classes/api/gkill-api"
import type { ApplicationConfig } from "../../classes/datas/config/application-config"
import type DnoteItem from "../../classes/dnote/dnote-item"

export default interface DnoteItemProps {
    gkill_api: GkillAPI
    application_config: ApplicationConfig
    editable: boolean
}