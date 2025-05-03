import type { GkillAPI } from "../../classes/api/gkill-api";
import type { ApplicationConfig } from "../../classes/datas/config/application-config";
import type AggregatedItem from "../../classes/dnote/aggregate-grouping-list-result-record";

export default interface AggregatedListItemProps {
    gkill_api: GkillAPI
    application_config: ApplicationConfig
    aggregated_item: AggregatedItem
}