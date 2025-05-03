import type { GkillAPI } from "@/classes/api/gkill-api"
import type { ApplicationConfig } from "@/classes/datas/config/application-config"

export default interface DnoteItemListViewProps {
    gkill_api: GkillAPI
    application_config: ApplicationConfig
    editable: boolean
}