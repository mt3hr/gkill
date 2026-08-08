'use strict'

import { i18n } from '@/i18n'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { GkillAPI } from '@/classes/api/gkill-api'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { ReKyou } from '@/classes/datas/re-kyou'
import type { MiReKyou } from '@/classes/datas/mi-re-kyou'
import { GetTagsByTargetIDRequest } from '@/classes/api/req_res/get-tags-by-target-id-request'
import { GetTextsByTargetIDRequest } from '@/classes/api/req_res/get-texts-by-target-id-request'
import { GetNotificationsByTargetIDRequest } from '@/classes/api/req_res/get-notifications-by-target-id-request'
import { GetReKyousByTargetIDRequest } from '@/classes/api/req_res/get-re-kyous-by-target-id-request'
import { GetMiReKyousByTargetIDRequest } from '@/classes/api/req_res/get-mi-re-kyous-by-target-id-request'
import { UpdateTagRequest } from '@/classes/api/req_res/update-tag-request'
import { UpdateTextRequest } from '@/classes/api/req_res/update-text-request'
import { UpdateNotificationRequest } from '@/classes/api/req_res/update-notification-request'
import { UpdateReKyouRequest } from '@/classes/api/req_res/update-re-kyou-request'
import { UpdateMiReKyouRequest } from '@/classes/api/req_res/update-mi-re-kyou-request'
import { UpdateKmemoRequest } from '@/classes/api/req_res/update-kmemo-request'
import { UpdateKCRequest } from '@/classes/api/req_res/update-kc-request'
import { UpdateURLogRequest } from '@/classes/api/req_res/update-ur-log-request'
import { UpdateNlogRequest } from '@/classes/api/req_res/update-nlog-request'
import { UpdateTimeisRequest } from '@/classes/api/req_res/update-timeis-request'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import { UpdateLantanaRequest } from '@/classes/api/req_res/update-lantana-request'
import { UpdateIDFKyouRequest } from '@/classes/api/req_res/update-idf-kyou-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

// 参照の連鎖を辿る深さの上限。
// サーバの逆引きが壊れたときに無限に掘り進まないための保険で、実データで届く値ではない
const max_cascade_depth = 32

// 1レベル内で同時に投げる逆引きの本数。1ノードにつき5リクエスト出るので、
// 幅の広い階層でファンアウトが爆発しないように区切る
const request_chunk_size = 16

export interface CascadeDeleteKyouOptions {
    kyou: Kyou
    gkill_api: GkillAPI
    application_config: ApplicationConfig
}

export interface CascadeDeleteKyouResult {
    // 画面から取り除くべきKyouのid。削除対象のKyou自身と、それを参照していたReKyou/MiReKyouのid
    deleted_ids: Array<string>
    errors: Array<GkillError>
}

export interface CascadeDeleteTargets {
    root_kyou: Kyou
    // 浅い順（rootに近い順）に積む。削除時は逆順に消す
    rekyous: Array<ReKyou>
    mirekyous: Array<MiReKyou>
    tags: Array<Tag>
    texts: Array<Text>
    notifications: Array<Notification>
    // root + 全ReKyou id + 全MiReKyou id
    visited_ids: Array<string>
    errors: Array<GkillError>
}

interface CascadeDeleteNode {
    tags: Array<Tag>
    texts: Array<Text>
    notifications: Array<Notification>
    rekyous: Array<ReKyou>
    mirekyous: Array<MiReKyou>
    errors: Array<GkillError>
}

interface UpdateStamp {
    is_deleted: boolean
    update_app: string
    update_device: string
    update_time: Date
    update_user: string
}

/**
 * Kyouと、それにくっついているTag/Text/Notification、それを参照しているReKyou/MiReKyouを
 * まとめて論理削除する。
 *
 * 探索（read）と削除（write）を完全に分ける。サーバのFindKyousは参照先が削除済みのReKyouを
 * 検索結果から外すので、参照先を消したあとでは辿れなくなる可能性があるため。
 * また削除が途中で失敗しても、対象のKyou自身が最後まで生きていれば同じダイアログをもう一度
 * 開くだけで残骸を再発見できる。追記型DAOなので再実行で収束する。
 */
export async function cascade_delete_kyou(options: CascadeDeleteKyouOptions): Promise<CascadeDeleteKyouResult> {
    const { kyou, gkill_api, application_config } = options

    // 共有画面はdeviceもuser_idも空なので、削除自体を行わない
    if (application_config.for_share_kyou) {
        return { deleted_ids: [], errors: [] }
    }

    await kyou.load_typed_datas()

    const targets = await discover_cascade_delete_targets(kyou, gkill_api)
    const mutate_errors = await mutate_cascade_delete_targets(targets, gkill_api, application_config)

    return {
        deleted_ids: targets.visited_ids,
        errors: targets.errors.concat(mutate_errors),
    }
}

/**
 * 消すべきものを幅優先で集める。updateは1本も投げない。
 *
 * 訪問済みidの集合で循環参照（A→B→A や自己参照）を止める。
 */
export async function discover_cascade_delete_targets(root_kyou: Kyou, gkill_api: GkillAPI): Promise<CascadeDeleteTargets> {
    const targets: CascadeDeleteTargets = {
        root_kyou: root_kyou,
        rekyous: new Array<ReKyou>(),
        mirekyous: new Array<MiReKyou>(),
        tags: new Array<Tag>(),
        texts: new Array<Text>(),
        notifications: new Array<Notification>(),
        visited_ids: new Array<string>(),
        errors: new Array<GkillError>(),
    }

    const visited = new Set<string>()
    visited.add(root_kyou.id)
    let frontier = [root_kyou.id]
    let depth = 0

    while (frontier.length !== 0) {
        if (depth > max_cascade_depth) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.cascade_delete_depth_exceeded
            error.error_message = i18n.global.t('CASCADE_DELETE_DEPTH_EXCEEDED_MESSAGE')
            targets.errors.push(error)
            break
        }

        const next_frontier = new Array<string>()
        for (let i = 0; i < frontier.length; i += request_chunk_size) {
            const chunk = frontier.slice(i, i + request_chunk_size)
            const nodes = await Promise.all(chunk.map(id => fetch_cascade_delete_node(id, gkill_api)))

            for (let j = 0; j < nodes.length; j++) {
                const node = nodes[j]
                targets.errors.push(...node.errors)
                targets.tags.push(...node.tags)
                targets.texts.push(...node.texts)
                targets.notifications.push(...node.notifications)

                for (const rekyou of dedupe_latest_by_id(node.rekyous)) {
                    if (rekyou.is_deleted || visited.has(rekyou.id)) {
                        continue
                    }
                    visited.add(rekyou.id)
                    targets.rekyous.push(rekyou)
                    next_frontier.push(rekyou.id)
                }
                for (const mirekyou of dedupe_latest_by_id(node.mirekyous)) {
                    if (mirekyou.is_deleted || visited.has(mirekyou.id)) {
                        continue
                    }
                    visited.add(mirekyou.id)
                    targets.mirekyous.push(mirekyou)
                    next_frontier.push(mirekyou.id)
                }
            }
        }

        frontier = next_frontier
        depth++
    }

    targets.visited_ids = Array.from(visited)
    return targets
}

/**
 * 1件のidについて、付随データと参照元をまとめて取ってくる。
 *
 * Tag/Text/NotificationはService Workerが target_id 単位でキャッシュしているので
 * force_reget を立てる。古い一覧のまま消すと取りこぼす。
 */
async function fetch_cascade_delete_node(id: string, gkill_api: GkillAPI): Promise<CascadeDeleteNode> {
    const tag_req = new GetTagsByTargetIDRequest()
    tag_req.target_id = id
    tag_req.force_reget = true

    const text_req = new GetTextsByTargetIDRequest()
    text_req.target_id = id
    text_req.force_reget = true

    const notification_req = new GetNotificationsByTargetIDRequest()
    notification_req.target_id = id
    notification_req.force_reget = true

    const rekyou_req = new GetReKyousByTargetIDRequest()
    rekyou_req.target_id = id

    const mirekyou_req = new GetMiReKyousByTargetIDRequest()
    mirekyou_req.target_id = id

    const [tag_res, text_res, notification_res, rekyou_res, mirekyou_res] = await Promise.all([
        gkill_api.get_tags_by_target_id(tag_req),
        gkill_api.get_texts_by_target_id(text_req),
        gkill_api.get_notifications_by_target_id(notification_req),
        gkill_api.get_rekyous_by_target_id(rekyou_req),
        gkill_api.get_mirekyous_by_target_id(mirekyou_req),
    ])

    return {
        tags: tag_res.tags ?? [],
        texts: text_res.texts ?? [],
        notifications: notification_res.notifications ?? [],
        rekyous: rekyou_res.rekyous ?? [],
        mirekyous: mirekyou_res.mirekyous ?? [],
        errors: new Array<GkillError>().concat(
            tag_res.errors ?? [],
            text_res.errors ?? [],
            notification_res.errors ?? [],
            rekyou_res.errors ?? [],
            mirekyou_res.errors ?? [],
        ),
    }
}

/**
 * 集めたものを実際に論理削除する。
 *
 * 1本失敗しても止めずに全部投げ、エラーは集約して返す。TXID/commit_txは使わない
 * （名前に反してDBトランザクションではなく部分確定しうるので、原子性は得られない）。
 *
 * update系のレスポンスは成功時 errors が null で来る（Goの構造体タグにomitemptyが無く、
 * nil sliceがそのまま "errors": null になる）。素のspreadはnullで例外を投げ、
 * 呼び出し元のダイアログクローズまで巻き添えにするので、必ず ?? [] を通す。
 */
async function mutate_cascade_delete_targets(targets: CascadeDeleteTargets, gkill_api: GkillAPI, application_config: ApplicationConfig): Promise<Array<GkillError>> {
    const errors = new Array<GkillError>()
    // 履歴のタイムスタンプがばらけないように、update_timeは全件で同じ値にする
    const stamp: UpdateStamp = {
        is_deleted: true,
        update_app: "gkill",
        update_device: application_config.device,
        update_time: new Date(Date.now()),
        update_user: application_config.user_id,
    }

    // 付随データ。親子関係がないので順序は問わない
    await Promise.all([
        ...targets.tags.map(async tag => {
            const req = new UpdateTagRequest()
            req.tag = Object.assign(tag.clone(), stamp)
            const res = await gkill_api.update_tag(req)
            errors.push(...(res.errors ?? []))
        }),
        ...targets.texts.map(async text => {
            const req = new UpdateTextRequest()
            req.text = Object.assign(text.clone(), stamp)
            const res = await gkill_api.update_text(req)
            errors.push(...(res.errors ?? []))
        }),
        ...targets.notifications.map(async notification => {
            const req = new UpdateNotificationRequest()
            req.notification = Object.assign(notification.clone(), stamp)
            const res = await gkill_api.update_notification(req)
            errors.push(...(res.errors ?? []))
        }),
    ])

    // 参照元。探索は終わっているので順序は必須ではないが、途中で失敗したときに
    // 「rootから辿れる形」をできるだけ残すため、深い方（rootから遠い方）から消す
    for (let i = targets.rekyous.length - 1; i >= 0; i--) {
        const req = new UpdateReKyouRequest()
        req.rekyou = Object.assign(targets.rekyous[i].clone(), stamp)
        const res = await gkill_api.update_rekyou(req)
        errors.push(...(res.errors ?? []))
    }
    for (let i = targets.mirekyous.length - 1; i >= 0; i--) {
        const req = new UpdateMiReKyouRequest()
        req.mirekyou = Object.assign(targets.mirekyous[i].clone(), stamp)
        const res = await gkill_api.update_mirekyou(req)
        errors.push(...(res.errors ?? []))
    }

    // Kyou自身は最後。先に消すとサーバのFindKyousが参照元を結果から外してしまい、
    // 途中で失敗したときに残骸を再発見できなくなる
    errors.push(...await delete_kyou_body(targets.root_kyou, gkill_api, stamp))

    // 消した全idのService Workerキャッシュを落とす
    await Promise.all(targets.visited_ids.map(id => delete_gkill_kyou_cache(id)))

    return errors
}

/**
 * Kyou自身のtyped dataに is_deleted を立てる。data_typeごとにエンドポイントが違う。
 *
 * 成功時のerrorsはnullで来るので ?? [] を通す（mutate_cascade_delete_targetsのコメント参照）。
 */
export async function delete_kyou_body(kyou: Kyou, gkill_api: GkillAPI, stamp: UpdateStamp): Promise<Array<GkillError>> {
    try {
        if (kyou.data_type.startsWith("kmemo")) {
            const req = new UpdateKmemoRequest()
            req.kmemo = Object.assign(kyou.typed_kmemo!.clone(), stamp)
            return (await gkill_api.update_kmemo(req)).errors ?? []
        }
        if (kyou.data_type.startsWith("kc")) {
            const req = new UpdateKCRequest()
            req.kc = Object.assign(kyou.typed_kc!.clone(), stamp)
            return (await gkill_api.update_kc(req)).errors ?? []
        }
        if (kyou.data_type.startsWith("urlog")) {
            const req = new UpdateURLogRequest()
            req.urlog = Object.assign(kyou.typed_urlog!.clone(), stamp)
            return (await gkill_api.update_urlog(req)).errors ?? []
        }
        if (kyou.data_type.startsWith("nlog")) {
            const req = new UpdateNlogRequest()
            req.nlog = Object.assign(kyou.typed_nlog!.clone(), stamp)
            return (await gkill_api.update_nlog(req)).errors ?? []
        }
        if (kyou.data_type.startsWith("timeis")) {
            const req = new UpdateTimeisRequest()
            req.timeis = Object.assign(kyou.typed_timeis!.clone(), stamp)
            return (await gkill_api.update_timeis(req)).errors ?? []
        }
        // mirekyou_* は "mi" で始まるためMiより先に判定し、Mi側からは除外する
        if (kyou.data_type.startsWith("mirekyou")) {
            const req = new UpdateMiReKyouRequest()
            req.mirekyou = Object.assign(kyou.typed_mirekyou!.clone(), stamp)
            return (await gkill_api.update_mirekyou(req)).errors ?? []
        }
        if (kyou.data_type.startsWith("mi")) {
            const req = new UpdateMiRequest()
            req.mi = Object.assign(kyou.typed_mi!.clone(), stamp)
            return (await gkill_api.update_mi(req)).errors ?? []
        }
        if (kyou.data_type.startsWith("lantana")) {
            const req = new UpdateLantanaRequest()
            req.lantana = Object.assign(kyou.typed_lantana!.clone(), stamp)
            return (await gkill_api.update_lantana(req)).errors ?? []
        }
        if (kyou.data_type.startsWith("idf")) {
            const req = new UpdateIDFKyouRequest()
            req.idf_kyou = Object.assign(kyou.typed_idf_kyou!.clone(), stamp)
            return (await gkill_api.update_idf_kyou(req)).errors ?? []
        }
        if (kyou.data_type.startsWith("rekyou")) {
            const req = new UpdateReKyouRequest()
            req.rekyou = Object.assign(kyou.typed_rekyou!.clone(), stamp)
            return (await gkill_api.update_rekyou(req)).errors ?? []
        }
        // git_commit_logは削除できない。ここに落ちたら未対応のdata_type
        return [build_cascade_delete_failed_error()]
    } catch (err: unknown) {
        // typed dataが読めていないなど。throwをそのまま上げるとダイアログが閉じないので
        // エラーに変換して返す
        console.error(err)
        return [build_cascade_delete_failed_error()]
    }
}

/**
 * 画面から取り除くためだけのKyou。消費側は id しか見ないので中身は詰めない。
 */
export function build_deleted_kyou_stub(id: string): Kyou {
    const kyou = new Kyou()
    kyou.id = id
    kyou.is_deleted = true
    return kyou
}

/**
 * 連鎖削除が想定外に失敗したときのエラー。呼び出し元の catch でも使う。
 */
export function build_cascade_delete_failed_error(): GkillError {
    const error = new GkillError()
    error.error_code = GkillErrorCodes.cascade_delete_failed
    error.error_message = i18n.global.t('FAILED_CASCADE_DELETE_KYOU_MESSAGE')
    return error
}

/**
 * 追記型DAOなので同じidの履歴が複数返りうる。update_timeが最大のものだけ残す。
 */
function dedupe_latest_by_id<T extends { id: string, update_time: Date }>(list: Array<T>): Array<T> {
    const by_id = new Map<string, T>()
    for (const item of list) {
        const prev = by_id.get(item.id)
        if (!prev || item.update_time.getTime() > prev.update_time.getTime()) {
            by_id.set(item.id, item)
        }
    }
    return Array.from(by_id.values())
}
