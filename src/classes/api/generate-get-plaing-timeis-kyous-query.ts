import moment from "moment"
import { FindKyouQuery } from "./find_query/find-kyou-query"
import { GkillAPI } from "./gkill-api"
import type { RepStructElementData } from "../datas/config/rep-struct-element-data"
import type { TagStructElementData } from "../datas/config/tag-struct-element-data"

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
        let rep_name_walk = (_rep: RepStructElementData): Array<string> => []
        rep_name_walk = (rep: RepStructElementData): Array<string> => {
            const rep_names = new Array<string>()
            const rep_children = rep.children
            if (rep_children) {
                rep_children.forEach(child_rep => {
                    rep_names.push(child_rep.rep_name)
                    if (child_rep.children) {
                        rep_names.push(...rep_name_walk(child_rep))
                    }
                })
            }
            return rep_names
        }
        plaing_timeis_query.reps = rep_name_walk(application_config.rep_struct)
        let tag_name_walk = (_tag: TagStructElementData): Array<string> => []
        tag_name_walk = (tag: TagStructElementData): Array<string> => {
            const tag_names = new Array<string>()
            const tag_children = tag.children
            if (tag_children) {
                tag_children.forEach(child_tag => {
                    if (child_tag.check_when_inited) {
                        tag_names.push(child_tag.tag_name)
                    }
                    if (child_tag.children) {
                        tag_names.push(...tag_name_walk(child_tag))
                    }
                })
            }
            return tag_names
        }
        plaing_timeis_query.tags = tag_name_walk(application_config.tag_struct)
    }
    return plaing_timeis_query
}