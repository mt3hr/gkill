'use strict'

import { watch } from 'vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { KyouChange, KyouChangeChannel, KyouChangeNotice } from '@/classes/kyou-change-bus'

/**
 * 画面ごとの受け口。
 *
 * **ここへ渡してよいのは emit を含まない適用関数だけ。**
 * 中継束（`crudRelayHandlers` 等）を渡すと、適用のたびに `emits(...)` が走って
 * ホストが再 publish し、通知が無限に往復する。
 */
export interface KyouChangeSink {
    /** 追加された Kyou を差し込む（局所挿入を持たない画面は再検索でよい） */
    apply_registered: (kyou: Kyou, requested_at: number) => void
    /** 1件を引き直す */
    apply_reload: (kyou: Kyou, requested_at: number) => void
    /** 一覧から取り除く */
    apply_deleted: (kyou: Kyou) => void
    /** 一覧を丸ごと取り直す */
    apply_reload_list: () => void
    /**
     * 他の画面で**新しく作られた**タグ名を、この画面の列の検索条件へ足す。
     *
     * **optional。** 列のタグ絞り込みを持たない画面（dashboard / plaing）は実装しない。
     * 受け手は「既知かどうか」を判定し直さないこと ―― 通知が届く頃には
     * 発生元の `check_tag_update` がタグツリーへ足し終えている可能性が高く、
     * やり直すと必ず取りこぼす。
     */
    apply_registered_tag?: (tag_names: Array<string>) => void
}

/**
 * 他のウィンドウで起きた変更を、この画面へ反映する。
 *
 * チャネルが `null`（＝単独ページ）なら購読も何も起きない。
 * 単独ページの実行時挙動はこれを足しても1バイトも変わらない。
 */
export function useKyouChangeSubscriber(channel: () => KyouChangeChannel | null, sink: KyouChangeSink): void {
    // 自分のカーソル。**購読開始時点の最大 seq から始める**。
    // 0 から始めると、後から開いたウィンドウが過去の変更を全部再生してしまう
    let consumed_seq = channel()?.bus.last_seq() ?? 0

    watch(() => channel()?.bus.last_seq() ?? 0, () => {
        const current = channel()
        if (!current) {
            return
        }
        const notices = current.bus.drain_from(consumed_seq)
        if (notices.length === 0) {
            return
        }
        consumed_seq = notices[notices.length - 1].seq
        apply_notices(current.origin_id, sink, notices)
    })
}

/**
 * 1回ぶんのドレインを適用する。
 *
 * `reload_list` は**1回に畳む**。畳まないと、KFTL のフォールバック1回で
 * 開いている画面ぶんの全件検索が走る。
 */
export function apply_notices(origin_id: string, sink: KyouChangeSink, notices: Array<KyouChangeNotice>): void {
    const mine = new Array<KyouChangeNotice>()
    const new_tag_names = new Array<string>()
    let needs_reload_list = false
    for (const notice of notices) {
        // 自分が出したものは受けない。受けると発生元が二重適用する
        // （追加は insert_kyou_sorted の id 重複判定で救われるが、削除と引き直しは救われない）
        if (notice.origin_id === origin_id) {
            continue
        }
        if (notice.change.kind === 'reload_list') {
            needs_reload_list = true
            continue
        }
        if (notice.change.kind === 'registered_tag') {
            new_tag_names.push(notice.change.tag_name)
            continue
        }
        mine.push(notice)
    }
    // **タグは reload_list の畳み込みより先に適用する。**
    // 条件を直す前に全件取り直しが走ると、旧条件のままなので記録が出ない
    if (new_tag_names.length !== 0) {
        sink.apply_registered_tag?.(new_tag_names)
    }
    if (needs_reload_list) {
        // どうせ全件取り直すので、個別の適用は無駄
        sink.apply_reload_list()
        return
    }
    for (const notice of mine) {
        const change: KyouChange = notice.change
        switch (change.kind) {
            case 'registered':
                sink.apply_registered(change.kyou, notice.requested_at)
                break
            case 'reload':
                sink.apply_reload(change.kyou, notice.requested_at)
                break
            case 'deleted':
                sink.apply_deleted(change.kyou)
                break
        }
    }
}
