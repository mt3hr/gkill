import moment from "moment"
import { FindKyouQuery } from "./find_query/find-kyou-query"
import { GkillAPI } from "./gkill-api"

export default function generate_get_plaing_timeis_kyous_query(fixed_time: Date | null): FindKyouQuery {
    const plaing_timeis_query = new FindKyouQuery()
    plaing_timeis_query.use_tags = false
    plaing_timeis_query.use_plaing = true
    plaing_timeis_query.plaing_time = moment().toDate()
    if (fixed_time && plaing_timeis_query.plaing_time.getTime() <= (fixed_time.getTime() ?? 0)) {
        plaing_timeis_query.plaing_time = moment(fixed_time.getTime()).add(1, 'second').toDate()
    } else {
        plaing_timeis_query.plaing_time = moment().toDate()
    }
    const application_config = GkillAPI.get_instance().get_saved_application_config()
    if (application_config) {
        application_config.rep_struct.forEach(rep_struct => {
            plaing_timeis_query.reps.push(rep_struct.rep_name)
        })
        application_config.tag_struct.forEach(tag_struct => {
            plaing_timeis_query.tags.push(tag_struct.tag_name)
        })
    }
    return plaing_timeis_query
}