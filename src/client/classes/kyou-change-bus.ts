// 編集前に読む: .claude/skills/gkill-client-columns/SKILL.md（この領域の不変条件の正本）
'use strict'

import { ref, shallowReactive } from 'vue'
import type { Kyou } from '@/classes/datas/kyou'

/**
 * 画面をまたいで配る変更の種類。
 *
 * `updated_kyou` と `requested_reload_kyou` は受け手の処理が同じなので
 * `reload` に統合してある。テキスト / 通知の追加・更新・削除は
 * 表示の更新としては `reload` に集約され、板ツリーの追随は
 * ページ側の `useConfigStructSync` が受け持つので、画面へ配る必要が無い。
 *
 * **タグだけは配る。** タグはツリーの追随だけでなく**列の検索条件にも効く**ので、
 * 配らないと「窓Aで新しいタグを付けて追加した記録が、窓Bでは条件に入らないまま
 * 一覧に出ず、しかも窓Bの列条件（枝番付きの localStorage）に焼き付く」が起きる。
 * 配るのは**発生元が「未知のタグだった」と判定したものだけ**で、
 * 受け手は判定をやり直さない（受け取る頃にはツリーへ足し終えている可能性が高く、
 * やり直すと必ず取りこぼす）。
 *
 * `requested_update_check_kyous` は**配らない** ―― 列ごとの選択状態であり、
 * rykv / mi では未実装（throw する）。
 */
export type KyouChange =
    | { kind: 'registered', kyou: Kyou }
    | { kind: 'reload', kyou: Kyou }
    | { kind: 'deleted', kyou: Kyou }
    | { kind: 'reload_list' }
    | { kind: 'registered_tag', tag_name: string }

export interface KyouChangeNotice {
    /** 単調増加。受け手は自分のカーソルより大きいものだけ消化する */
    seq: number
    /** 発生元のウィンドウ id。自分が出したものは受けない */
    origin_id: string
    /**
     * `new_reload_batch()` の値。**発生元が採番**して自分のローカル適用と共有する。
     * これが揃っていると `kyou-reload.ts` の合流が効いて、画面が何枚あっても
     * `/api/get_kyou` は1往復で済む
     */
    requested_at: number
    change: KyouChange
}

/**
 * 保持する通知の上限。
 * 受け手は自分のカーソルから消化するので、取りこぼさない限り数件しか溜まらない。
 * 想定外に溜まったときのメモリ青天井を防ぐためだけの数字
 */
const KYOU_CHANGE_LOG_MAX_LENGTH = 200

export interface KyouChangeBus {
    /** 追記専用ログ。古いものから捨てる */
    log: Array<KyouChangeNotice>
    /**
     * 変更検知の起点。受け手はこれを watch のソースで読む。
     *
     * **Ref をそのまま公開してはいけない。** このオブジェクトが reactive() に
     * 包まれると Vue がプロパティ読み出しで Ref を自動アンラップするので、
     *  が undefined になり、**伝播が黙って効かなくなる**。
     * 関数にしておけば包まれ方に関係なく動く
     */
    last_seq(): number
    publish(origin_id: string, change: KyouChange, requested_at: number): void
    /** `seq` より大きい通知を古い順に返す */
    drain_from(seq: number): Array<KyouChangeNotice>
}

/**
 * 画面間の変更通知バス。ポート(rudbeckia)のページが1つだけ作る。
 *
 * **スカラー（最新の1件）ではなく追記ログにしてある。** 同じ tick に複数件起きると
 * スカラーでは最後の1件しか見えず、残りが黙って落ちる（KFTLの複数行保存が典型）。
 *
 * バスは props で配る。`provide` / `inject` にしてはいけない ―― 既存のテストは
 * `useRykvView({props, emits})` をコンポーネントインスタンスの外から素で呼ぶので、
 * `inject()` は警告を出して既定値へ落ちる。つまり**テストでは伝播が効かないのに緑**になる。
 */
export function create_kyou_change_bus(): KyouChangeBus {
    // 中身の入れ替えは起きず push / shift だけなので shallow で足りる
    const log = shallowReactive(new Array<KyouChangeNotice>())
    const last_seq_ref = ref(0)
    let seq_counter = 0

    function publish(origin_id: string, change: KyouChange, requested_at: number): void {
        seq_counter++
        log.push({
            seq: seq_counter,
            origin_id: origin_id,
            requested_at: requested_at,
            change: change,
        })
        while (log.length > KYOU_CHANGE_LOG_MAX_LENGTH) {
            log.shift()
        }
        last_seq_ref.value = seq_counter
    }

    function drain_from(seq: number): Array<KyouChangeNotice> {
        const drained = new Array<KyouChangeNotice>()
        for (const notice of log) {
            if (notice.seq > seq) {
                drained.push(notice)
            }
        }
        return drained
    }

    return {
        log: log,
        last_seq: () => last_seq_ref.value,
        publish: publish,
        drain_from: drain_from,
    }
}

/**
 * 1つのウィンドウがバスへ繋がるための一式。ビューへは**この1つだけ**を prop で渡す。
 *
 * `null` が「単独ページとして動く」＝ publish も購読もしない。
 * `FindQuery` の「null = そのフィルタ未使用」と同じ約束で、`undefined` は使わない。
 */
export interface KyouChangeChannel {
    bus: KyouChangeBus
    /** このウィンドウの id。自分が出した通知を受けないための印 */
    origin_id: string
}

export { KYOU_CHANGE_LOG_MAX_LENGTH }
