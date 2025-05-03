import type { GkillAPI } from "@/classes/api/gkill-api";
import type { ApplicationConfig } from "@/classes/datas/config/application-config";
import type DnoteListQuery from "@/pages/views/dnote-list-query";

export default interface EditDnoteListViewProps {
    gkill_api: GkillAPI
    application_config: ApplicationConfig
    dnote_list_query: DnoteListQuery
}