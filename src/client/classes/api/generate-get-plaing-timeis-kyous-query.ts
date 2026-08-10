import moment from "moment"
import { FindKyouQuery } from "./find_query/find-kyou-query"
import { generate_plaing_timeis_query } from "./find_query/generate-plaing-timeis-query"
import { GkillAPI } from "./gkill-api"

// 実行中画面とKFTLの/end系終了候補検索が使うplaing検索クエリを生成する。
// 基準時刻は現在時刻（fixed_timeが未来を指すときはfixed_time+1秒）。
// 検索条件の組み立ては generate_plaing_timeis_query に委譲しており、
// ApplicationConfigのカスタム検索条件（plaing_timeis_json_data）もそこで適用される。
export default function generate_get_plaing_timeis_kyous_query(fixed_time: Date | null): FindKyouQuery {
    let plaing_time = moment().toDate()
    if (fixed_time && plaing_time.getTime() <= fixed_time.getTime()) {
        plaing_time = moment(fixed_time.getTime()).add(1, 'second').toDate()
    }
    const application_config = GkillAPI.get_instance().get_saved_application_config()
    return generate_plaing_timeis_query(application_config, plaing_time)
}
