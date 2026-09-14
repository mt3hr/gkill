import moment from "moment"
import { FindKyouQuery } from "./find_query/find-kyou-query"
import { generate_playing_timeis_query } from "./find_query/generate-playing-timeis-query"
import { GkillAPI } from "./gkill-api"

// 実行中画面が使うplaying検索クエリを生成する（KFTLの/end系の対象検索はサーバ側 kftl_timeis.go が同じ条件で行う）。
// 基準時刻は現在時刻（fixed_timeが未来を指すときはfixed_time+1秒）。
// 検索条件の組み立ては generate_playing_timeis_query に委譲しており、
// ApplicationConfigのカスタム検索条件（playing_timeis_json_data）もそこで適用される。
export default function generate_get_playing_timeis_kyous_query(fixed_time: Date | null): FindKyouQuery {
    let playing_time = moment().toDate()
    if (fixed_time && playing_time.getTime() <= fixed_time.getTime()) {
        playing_time = moment(fixed_time.getTime()).add(1, 'second').toDate()
    }
    const application_config = GkillAPI.get_instance().get_saved_application_config()
    return generate_playing_timeis_query(application_config, playing_time)
}
