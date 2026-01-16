import type { GkillAPI } from "../../classes/api/gkill-api"
import type { ApplicationConfig } from "../../classes/datas/config/application-config"

export default interface DnoteItemProps {
    gkill_api: GkillAPI
    application_config: ApplicationConfig
    editable: boolean
    dnd_list_index: number
}