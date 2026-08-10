'use strict'

import type { Ref } from 'vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { GkillAPI } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import { GetAllTagNamesRequest } from '@/classes/api/req_res/get-all-tag-names-request'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import { board_exists_in_mi_board_struct } from '@/classes/mi-board-struct'
import { tag_exists_in_tag_struct } from '@/classes/tag-struct'

/**
 * 板ツリー・タグツリーをセッション中に最新へ追随させる。
 *
 * 板もタグもサーバに実体が無く、一覧API(`get_mi_board_list` / `get_all_tag_names`)から
 * `ApplicationConfig.load_all()` が起動時にツリーへ流し込んでいる導出概念。
 * つまり再読込すれば必ず最新になるので、ここでやることは
 * 「一覧のキャッシュを force_reget で更新 → ツリーへ足す → clone() で反映」
 * という純粋なフロント処理だけ。サーバへは何も書かない。
 *
 * 以前は use-mi-page.ts だけが板とタグの両方を持ち、タグだけが rykv / plaing / mkfl に
 * コピーされていて、dashboard / kyou / saihate / kftl には何も無かった。
 * 追随処理を持たないページで板やタグを増やすと、その画面のツリーと
 * 板名ドロップダウンが再読込まで古いままになる。
 */
export function useConfigStructSync(options: {
    application_config: Ref<ApplicationConfig>,
    gkill_api: () => GkillAPI,
    write_errors: (errors: Array<GkillError>) => void,
}) {
    const { application_config, gkill_api, write_errors } = options

    // ── 連打/連続登録で二重に通信しないため ──
    let tag_struct_refresh_promise: Promise<void> | null = null
    let mi_board_struct_refresh_promise: Promise<void> | null = null

    // ── Internal helpers ──
    /**
     * タグツリーへ未登録のタグを流し込む。
     * `append_not_found_tags()` はルート直下へ push するので、増減はルートの children 数で判る。
     * 増えていないときに clone() すると、identity 変化を見ている watch が無駄に走るので避ける。
     */
    async function refresh_tag_struct(): Promise<void> {
        // すでに更新中ならそれに乗る
        if (tag_struct_refresh_promise) {
            await tag_struct_refresh_promise
            return
        }

        tag_struct_refresh_promise = (async () => {
            const before = application_config.value.tag_struct.children?.length ?? 0
            const errors = await application_config.value.append_not_found_tags()
            if (errors && errors.length) {
                write_errors(errors)
                return
            }
            const after = application_config.value.tag_struct.children?.length ?? 0
            if (before === after) {
                return
            }

            // 深い変更だけでは `watch(() => props.application_config, ...)` が発火しないので、
            // identity を変えて配下のフォームに引き直させる
            application_config.value = application_config.value.clone()

            gkill_api().set_saved_application_config(application_config.value)
        })()

        try {
            await tag_struct_refresh_promise
        } finally {
            tag_struct_refresh_promise = null
        }
    }

    /** 板ツリー版。手順は refresh_tag_struct と同じ */
    async function refresh_mi_board_struct(): Promise<void> {
        if (mi_board_struct_refresh_promise) {
            await mi_board_struct_refresh_promise
            return
        }

        mi_board_struct_refresh_promise = (async () => {
            const before = application_config.value.mi_board_struct.children?.length ?? 0
            const errors = await application_config.value.append_not_found_mi_boards()
            if (errors && errors.length) {
                write_errors(errors)
                return
            }
            const after = application_config.value.mi_board_struct.children?.length ?? 0
            if (before === after) {
                return
            }

            application_config.value = application_config.value.clone()

            gkill_api().set_saved_application_config(application_config.value)
        })()

        try {
            await mi_board_struct_refresh_promise
        } finally {
            mi_board_struct_refresh_promise = null
        }
    }

    // ── Business logic ──
    /** 登録・更新されたタグがツリーに無ければ足す */
    async function check_tag_update(tag: Tag): Promise<void> {
        const name = tag.tag
        if (!name) return

        const req = new GetAllTagNamesRequest()
        req.force_reget = true
        await gkill_api().get_all_tag_names(req)

        if (tag_exists_in_tag_struct(name, application_config.value.tag_struct)) return

        await refresh_tag_struct()
    }

    /** 登録・更新されたKyouが持つ板名がツリーに無ければ足す */
    async function check_mi_board_update(kyou: Kyou): Promise<void> {
        // 板を持つのは Mi と MiReKyou だけ。それ以外は get_kyou を投げる前に弾く
        // （data_type が空のときは判断できないので通す）
        const data_type = kyou.data_type ?? ""
        if (data_type !== "" && !data_type.startsWith("mi")) return

        const get_kyou_req = new GetKyouRequest()
        get_kyou_req.id = kyou.id
        const get_kyou_res = await gkill_api().get_kyou(get_kyou_req)
        if (!get_kyou_res.kyou_histories || get_kyou_res.kyou_histories.length === 0) {
            return
        }
        const latest_kyou = get_kyou_res.kyou_histories[0]

        // "mirekyou" は "mi" に前方一致するので、MiReKyou を先に判定すること。
        // 以前は typed_mi しか見ておらず、MiReKyou で入力した板名を取りこぼしていた
        let name = ""
        if ((latest_kyou.data_type ?? "").startsWith("mirekyou")) {
            await latest_kyou.load_typed_mirekyou()
            name = latest_kyou.typed_mirekyou ? latest_kyou.typed_mirekyou.board_name : ""
        } else {
            await latest_kyou.load_typed_mi()
            name = latest_kyou.typed_mi ? latest_kyou.typed_mi.board_name : ""
        }
        if (!name) return

        const req = new GetMiBoardRequest()
        req.force_reget = true
        await gkill_api().get_mi_board_list(req)

        if (board_exists_in_mi_board_struct(name, application_config.value.mi_board_struct)) return

        await refresh_mi_board_struct()
    }

    /**
     * 板・タグの両方を取り直して足す。
     * KFTL は保存したタグを `registered_tag` で上げてこない（KFTLViewEmits に無い）ので、
     * 保存完了の合図 `saved_kyou_by_kftl` を受けたページはこちらを使う。
     */
    async function resync_structs(): Promise<void> {
        const tag_req = new GetAllTagNamesRequest()
        tag_req.force_reget = true
        await gkill_api().get_all_tag_names(tag_req)

        const board_req = new GetMiBoardRequest()
        board_req.force_reget = true
        await gkill_api().get_mi_board_list(board_req)

        await refresh_tag_struct()
        await refresh_mi_board_struct()
    }

    // ── Return ──
    return {
        check_tag_update,
        check_mi_board_update,
        resync_structs,
    }
}
