import { pruneEmpty } from "./export_prune"

export function toExportKyouDto(kyou: any) {
    const meta = {
        // id: kyou.id,
        data_type: kyou.data_type,
        related_time: toIso(kyou.related_time),
        // create_time: toIso(kyou.create_time),
        // update_time: toIso(kyou.update_time),
        // image_source: kyou.image_source,
    }

    const tags =
        kyou.attached_tags?.map((t: any) => t.tag)

    const texts =
        kyou.attached_texts?.map((t: any) => t.text)

    const notifications =
        kyou.attached_notifications?.map((n: any) => ({
            content: n.content,
            notification_time: toIso(n.notification_time),
            is_notificated: n.is_notificated,
        }))

    const payload =
        kyou.typed_timeis ? {
            kind: "timeis",
            title: kyou.typed_timeis.title,
            start_time: toIso(kyou.typed_timeis.start_time),
            end_time: toIso(kyou.typed_timeis.end_time),
        } :
            kyou.typed_kmemo ? {
                kind: "kmemo",
                content: kyou.typed_kmemo.content,
            } :
                kyou.typed_kc ? {
                    kind: "kc",
                    key: kyou.typed_kc.key,
                    value: kyou.typed_kc.value,
                } :
                    kyou.typed_urlog ? {
                        kind: "urlog",
                        title: kyou.typed_urlog.title,
                        url: kyou.typed_urlog.url,
                    } :
                        kyou.typed_nlog ? {
                            kind: "nlog",
                            title: kyou.typed_nlog.title,
                            shop: kyou.typed_nlog.shop,
                            amount: kyou.typed_nlog.amount,
                        } :
                            kyou.typed_mi ? {
                                kind: "mi",
                                title: kyou.typed_mi.title,
                                is_checked: kyou.typed_mi.is_checked,
                                board_name: kyou.typed_mi.board_name,
                                limit_time: toIso(kyou.typed_mi.limit_time),
                                estimate_start_time: toIso(kyou.typed_mi.estimate_start_time),
                                estimate_end_time: toIso(kyou.typed_mi.estimate_end_time),
                            } :
                                kyou.typed_lantana ? {
                                    kind: "lantana",
                                    mood: kyou.typed_lantana.mood,
                                } :
                                    kyou.typed_idf_kyou ? {
                                        kind: "idf",
                                        // file_path: kyou.typed_idf_kyou.file_path,
                                    } :
                                        kyou.typed_git_commit_log ? {
                                            kind: "git",
                                            commit_message: kyou.typed_git_commit_log.commit_message,
                                            addition: kyou.typed_git_commit_log.addition,
                                            deletion: kyou.typed_git_commit_log.deletion,
                                        } :
                                            // kyou.typed_rekyou ? {
                                            // kind: "rekyou",
                                            // from_id: kyou.typed_rekyou.from_id,
                                            // to_id: kyou.typed_rekyou.to_id,
                                            // } :
                                            undefined

    return pruneEmpty({
        meta,
        tags,
        texts,
        notifications,
        payload,
    })
}

function toIso(v: any): string | undefined {
    if (!v) return undefined
    if (v instanceof Date) return v.toISOString()
    return String(v)
}