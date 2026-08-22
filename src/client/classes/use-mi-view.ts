
// 列の同一性を query_id に固定する理由と、再採番が誤配送を招く経緯:
// documents/adr/0034-column-identity-query-id.md
import { log_unless_aborted } from '@/classes/abort-error'
import { gkill_page_list } from '@/classes/gkill-page-list'
import router from '@/router'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { computed, nextTick, onBeforeUnmount, type Ref, ref, toRaw, watch } from 'vue'
import { Kyou } from '@/classes/datas/kyou'
import type { MiViewEmits } from '@/pages/views/mi-view-emits'
import type { MiViewProps } from '@/pages/views/mi-view-props'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import { GetKyousResponse } from '@/classes/api/req_res/get-kyous-response'
import moment from 'moment'
import { deep_equals } from '@/classes/deep-equals'
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'
import { useScopedCtrlVForClipboard } from '@/classes/use-scoped-ctrl-v-for-clipboard'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { Tag } from '@/classes/datas/tag'
import { Mi } from '@/classes/datas/mi'
import { MiReKyou } from '@/classes/datas/mi-re-kyou'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import { UpdateMiReKyouRequest } from '@/classes/api/req_res/update-mi-re-kyou-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import { new_reload_batch, refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import type { KyouChange } from '@/classes/kyou-change-bus'
import { useKyouChangeSubscriber } from '@/classes/use-kyou-change-subscriber'
import { useRegisteredKyouLocalInsert } from '@/classes/use-registered-kyou-local-insert'
import { useRegisteredTagColumnFilter } from '@/classes/use-registered-tag-column-filter'
import { tag_exists_in_tag_struct } from '@/classes/tag-struct'
import { remove_kyou_from_list_by_id, apply_mi_projection, insert_kyou_sorted } from '@/classes/kyou-local-insert'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { ComponentRef } from '@/classes/component-ref'
import { MI_ALL_BOARD_KEY } from '@/classes/mi-board-names'

// ドラッグ&ドロップで受け取ったJSONからMiに写してよいフィールド。
// 内容は Mi.clone() が複製しているフィールドと同じ
const MI_DROP_ALLOWED_KEYS = new Set([
    "is_deleted", "id", "rep_name", "related_time", "data_type",
    "create_time", "create_app", "create_device", "create_user",
    "update_time", "update_app", "update_user", "update_device",
    "title", "is_checked", "board_name",
    "limit_time", "estimate_start_time", "estimate_end_time",
])

// MiReKyou版。MiReKyouはtitleを持たず、代わりにtarget_idを持つ
const MI_REKYOU_DROP_ALLOWED_KEYS = new Set([
    "is_deleted", "id", "rep_name", "related_time", "data_type",
    "create_time", "create_app", "create_device", "create_user",
    "update_time", "update_app", "update_user", "update_device",
    "target_id", "is_checked", "board_name",
    "limit_time", "estimate_start_time", "estimate_end_time",
])

// DataTransferのJSONは外部由来なので、許可したフィールド以外は捨てる
function parse_dropped_task<T extends object>(json: unknown, instance: T, allowed_keys: Set<string>): T {
    if (json === null || typeof json !== 'object') {
        return instance
    }
    // 許可キーだけを写す。値の型までは分からないので、
    // 書き込み口だけを Record<string, unknown> として扱う（any は使わない）
    const source = json as Record<string, unknown>
    const target = instance as unknown as Record<string, unknown>
    for (const key of Object.keys(source)) {
        if (!allowed_keys.has(key)) {
            continue
        }
        target[key] = source[key]

        // 時刻はDate型に変換
        const value = target[key]
        if (key.endsWith("time") && value) {
            target[key] = new Date(value as string | number | Date)
        }
    }
    return instance
}

export function useMiView(options: {
    props: MiViewProps,
    emits: MiViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const mi_root = ref<HTMLElement | null>(null)
    const query_editor_sidebar = ref<ComponentRef | null>(null)
    const add_mi_dialog = ref<ComponentRef | null>(null)
    const add_nlog_dialog = ref<ComponentRef | null>(null)
    const add_lantana_dialog = ref<ComponentRef | null>(null)
    const add_timeis_dialog = ref<ComponentRef | null>(null)
    const add_urlog_dialog = ref<ComponentRef | null>(null)
    const kftl_dialog = ref<ComponentRef | null>(null)
    const add_kc_dialog = ref<ComponentRef | null>(null)
    const mkfl_dialog = ref<ComponentRef | null>(null)
    const upload_file_dialog = ref<ComponentRef | null>(null)
    const save_clipboard_to_file_dialog = ref<ComponentRef | null>(null)
    // 列ごとに1つ。v-for の ref なので配列で来る
    const kyou_list_views = ref<Array<ComponentRef> | null>(null)

    // ── State refs ──
    const enable_context_menu = ref(true)
    const enable_dialog = ref(true)
    const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])
    // 板のドラッグ&ドロップ移動が進行中か
    const is_moving_board_task = ref(false)

    const querys: Ref<Array<FindKyouQuery>> = ref([new FindKyouQuery()])
    const querys_backup: Ref<Array<FindKyouQuery>> = ref(new Array<FindKyouQuery>()) // 更新検知用バックアップ
    const match_kyous_list: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
    const match_kyous_list_top_list: Ref<Array<number>> = ref(new Array<number>())
    const focused_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const focused_column_index: Ref<number> = ref(0)
    const focused_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const focused_kyou: Ref<Kyou | null> = ref(null)
    const focused_time: Ref<Date> = ref(moment().toDate())
    const is_show_kyou_detail_view: Ref<boolean> = ref(false)
    const is_show_kyou_count_calendar: Ref<boolean> = ref(false)
    const drawer: Ref<boolean | null> = ref(false)
    // 一時表示(オーバーレイ)モードにするかは「今の内容領域の幅」で決まる。
    // 初期化時に1回代入するだけにしてはいけない ―― このビューはポート(rudbeckia)の
    // リサイズできるダイアログの中でも描かれるので、幅はあとから変わる
    const drawer_mode_is_mobile = computed<boolean>(() => !(props.app_content_width.valueOf() >= 760))
    const is_loading: Ref<boolean> = ref(true)
    const inited = ref(false)
    const is_restoring_columns = ref(false) // 保存済み列の初期検索がまだ走っている。表示制御には使わない
    const running_search_count = ref(0) // 飛行中の検索の本数。準備完了信号にだけ使う
    const received_init_request = ref(false)
    const skip_search_this_tick = ref(false)
    const abort_controllers = new Map<string, AbortController>() // 列ごとの検索中断用。キーは列のquery_id
    const search_seqs = new Map<string, number>() // 列ごとの検索の世代番号。キーは列のquery_id。最後の検索だけが結果を書き戻せるようにする

    // ── Computed ──
    const kyou_list_view_height = computed(() => props.app_content_height)

    // 「操作してよい状態」をE2Eが決定論的に待つための信号。
    // 画面の表示/非表示には一切使わない(使うと初期検索を待たなくした意味がなくなる)
    const is_view_ready = computed(() =>
        inited.value && !is_restoring_columns.value && running_search_count.value === 0)

    // 画面切替メニューの一覧は classes/gkill-page-list.ts に1つだけ置いてある
    const page_list = gkill_page_list

    // ── Init trigger ──
    // ApplicationConfig が来たことが初期化の唯一の前提条件なので、それを直接待つ。
    // 以前はサイドバーの @inited を起動条件にしていたが、あれは子ビューの
    // 「その節が描けた」の集約でしかない。設定の到着を表していたのは
    // 「immediateの付いていない application_config watch から emit する子がいる」
    // という偶然で、mi では実質 CalendarQuery 1つが律速していた(しかもその節は
    // application_config のフィールドを1つも読まない)。
    // immediate は「setup時点で既にロード済み」の将来ケースへの保険
    watch(() => props.application_config.is_loaded, (is_loaded: boolean) => {
        if (!is_loaded) {
            return
        }
        if (received_init_request.value) {
            return
        }
        received_init_request.value = true
        init()
    }, { immediate: true })

    // ── Lifecycle ──
    onBeforeUnmount(() => {
        // 飛行中の検索を止める。ポート(rudbeckia)ではこのビューごと閉じられるので、
        // 放っておくと閉じたウィンドウぶんの数十万件の取得が最後まで走る
        for (const abort_controller of abort_controllers.values()) {
            abort_controller.abort()
        }
        abort_controllers.clear()
    })

    // ── Watchers ──
    watch(() => is_show_kyou_count_calendar.value, () => {
        if (is_show_kyou_count_calendar.value) {
            update_focused_kyous_list(focused_column_index.value)
        }
    })

    watch(() => focused_time.value, () => {
        // 初期化前は初期クエリのquery_idが空文字のままなので、
        // 「列が無い」判定はquery_idの真偽値ではなく列の存在で行う
        const focused_column_query = querys.value[focused_column_index.value]
        if (!focused_column_query) {
            return
        }
        const kyou_list_view = get_kyou_list_view(focused_column_query.query_id)
        if (!kyou_list_view) {
            return
        }
        let target_kyou: Kyou | null = null
        for (let i = 0; i < focused_kyous_list.value.length; i++) {
            const kyou = focused_kyous_list.value[i]
            if (kyou.related_time.getTime() >= focused_time.value.getTime()) {
                target_kyou = kyou
                break
            }
        }
        if (inited.value) {
            kyou_list_view.scroll_to_kyou(target_kyou)
        }
    })

    // ── Internal helpers ──
    function update_focused_kyous_list(column_index: number): void {
        if (!match_kyous_list.value || match_kyous_list.value.length === 0) {
            return
        }
        focused_kyous_list.value = match_kyous_list.value[column_index]
    }

    // 列コンポーネントをquery_idで引く。v-forのテンプレートref配列はマウント順で
    // 並び順が保証されないため、indexではなくquery_idで解決する
    function get_kyou_list_view(query_id: string): ComponentRef | undefined {
        return kyou_list_views.value?.filter((kyou_list_view) => kyou_list_view.get_query_id() === query_id)[0]
    }

    // フォーカス切り替え等でfocused_queryを差し替えると、サイドバーが機械的に
    // updated_queryを発火しうるので、差し替えを行うfnをこれで包んで1回分だけ検索を抑止する。
    // 抑止解除のnextTickは「fnのリアクティブ書き込みでflushが予約された後」に登録すること。
    // 書き込みより先に登録すると、VueのnextTickはresolvedPromiseへ直結し、
    // 解除がウォッチャflushより先に走って抑止が一度も効かない(マイクロタスクはFIFO)。
    // コールバック式はこの登録順を構造的に保証する。
    // 立てっぱなしにするとユーザの次の編集を黙殺するため、必ずnextTickで倒す
    function run_with_sidebar_search_suppressed(fn: () => void): void {
        skip_search_this_tick.value = true
        fn()
        nextTick(() => skip_search_this_tick.value = false)
    }

    function remove_kyou_from_multi_column_lists(lists: Array<Array<Kyou>>, deleted_id: string): void {
        for (let i = 0; i < lists.length; i++) {
            remove_kyou_from_list_by_id(lists[i], deleted_id)
        }
    }

    function is_target_mi_kyou(kyou: Kyou, mi_id: string): boolean {
        return kyou.typed_mi?.id === mi_id || kyou.typed_mirekyou?.id === mi_id || kyou.id === mi_id
    }

    function find_kyou_instances_by_mi_id(mi_id: string): Array<{ column_index: number, row_index: number, kyou: Kyou }> {
        const instances: Array<{ column_index: number, row_index: number, kyou: Kyou }> = []
        for (let column_index = 0; column_index < match_kyous_list.value.length; column_index++) {
            const column = match_kyous_list.value[column_index]
            for (let row_index = 0; row_index < column.length; row_index++) {
                const kyou = column[row_index]
                if (is_target_mi_kyou(kyou, mi_id)) {
                    instances.push({ column_index, row_index, kyou })
                }
            }
        }
        return instances
    }

    function remove_kyou_from_column_by_id(column_index: number, kyou_id: string): void {
        const column = match_kyous_list.value[column_index]
        if (!column) {
            return
        }
        for (let i = column.length - 1; i >= 0; i--) {
            if (column[i].id === kyou_id) {
                column.splice(i, 1)
            }
        }
    }

    // 移動先の列へ、その列の並び替え規則で差し込む。
    // 移動元の行の related_time / data_type は「移動元の列の mi_sort_type」で計算されており、
    // 移動先と一致するとは限らないので、射影を計算し直してから入れる
    function insert_kyou_into_column_if_absent(column_index: number, kyou: Kyou): void {
        const column = match_kyous_list.value[column_index]
        const column_query = querys.value[column_index]
        if (!column || !column_query) {
            return
        }
        const inserting_kyou = kyou.clone()
        apply_mi_projection(inserting_kyou, column_query.mi_sort_type)
        insert_kyou_sorted(column, inserting_kyou, column_query)
    }

    function patch_kyou_mi_board_name(kyou: Kyou, updated_mi: Mi | MiReKyou): void {
        // MiReKyouのKyouにはtyped_mirekyouが載っているのでそちらを更新する
        const target = (updated_mi instanceof MiReKyou || kyou.typed_mirekyou)
            ? (kyou.typed_mirekyou ??= new MiReKyou())
            : (kyou.typed_mi ??= new Mi())
        target.id = updated_mi.id
        target.board_name = updated_mi.board_name
        target.update_app = updated_mi.update_app
        target.update_device = updated_mi.update_device
        target.update_user = updated_mi.update_user
        target.update_time = updated_mi.update_time
    }

    function apply_board_move_locally(mi_id: string, before_board: string, after_board: string, updated_mi: Mi | MiReKyou): void {
        const instances = find_kyou_instances_by_mi_id(mi_id)
        if (instances.length === 0) {
            return
        }

        // 既存インスタンスにボード更新を反映
        for (let i = 0; i < instances.length; i++) {
            patch_kyou_mi_board_name(instances[i].kyou, updated_mi)
        }
        const target_kyou = instances[0].kyou

        for (let i = 0; i < querys.value.length; i++) {
            const query = querys.value[i]
            if (query.mi_board_name !== null && query.mi_board_name === before_board) {
                remove_kyou_from_column_by_id(i, target_kyou.id)
            }
            if (query.mi_board_name !== null && query.mi_board_name === after_board) {
                insert_kyou_into_column_if_absent(i, target_kyou)
            }
        }

        if (focused_kyou.value && is_target_mi_kyou(focused_kyou.value, mi_id)) {
            patch_kyou_mi_board_name(focused_kyou.value, updated_mi)
        }
        if (is_show_kyou_count_calendar.value) {
            update_focused_kyous_list(focused_column_index.value)
        }
    }

    // ── Business logic ──
    /**
     * 一覧から取り除くだけ。**emit を含めないこと** ―― ポート(rudbeckia)の
     * 変更通知はこの関数を直接呼ぶので、ここで emit するとホストが再 publish して
     * 通知が無限に往復する
     */
    function apply_deleted_kyou(deleted_kyou: Kyou): void {
        remove_kyou_from_multi_column_lists(match_kyous_list.value, deleted_kyou.id)
        remove_kyou_from_list_by_id(focused_kyous_list.value, deleted_kyou.id)
        if (focused_kyou.value?.id === deleted_kyou.id) {
            focused_kyou.value = null
        }
    }

    function onDeletedKyou(deleted_kyou: Kyou): void {
        apply_deleted_kyou(deleted_kyou)
        emits('deleted_kyou', deleted_kyou)
    }

    // 対象のKyouが載っている列のクエリを探す。focused_kyou と opened_dialogs は
    // 自分がどの列由来か知らないので、列を総当たりして最初に見つかったものを使う
    // その Kyou が乗っている列の検索条件を返す。
    //
    // rykv の対になる関数は `find_reload_query_for(kyou_id, data_type)` で、
    // あちらは列のクエリが Mi 用ではない（for_mi=false）ため
    // `build_mi_reload_query` で for_mi / mi_sort_type を足してから返す。
    // mi の列のクエリは最初から Mi 用なので、ここは列のクエリをそのまま返す。
    //
    // 走査は生の配列に対して行う。match_kyous_list は deep なリアクティブ ref なので、
    // 素で `.some()` すると1要素ごとに track と Proxy 生成が走る
    // （ダイアログを開くたび・引き直すたびに列を総当たりするので効く）。
    function find_column_query_for(kyou_id: string): FindKyouQuery | undefined {
        for (let i = 0; i < match_kyous_list.value.length; i++) {
            const raw_list = toRaw(match_kyous_list.value[i])
            for (let j = 0; j < raw_list.length; j++) {
                if (raw_list[j].id === kyou_id) {
                    return querys.value[i]
                }
            }
        }
        return undefined
    }

    /**
     * @param requested_at_arg 引き直しの合流キー。ポートの変更通知から呼ぶときは
     *   **発生元が採番した値**を渡す。渡さないとここで採番され、
     *   kyou-reload.ts の合流が成立せず画面の枚数ぶん往復する
     */
    async function reload_kyou(kyou: Kyou, requested_at_arg?: number): Promise<void> {
        // 列・focused・開いているダイアログは同じ更新を受けて独立に引き直す。
        // 同じ値を渡して1往復に合流させる(渡さないと系統ごとに往復が増える)
        const requested_at = requested_at_arg ?? new_reload_batch();
        (async (): Promise<void> => {
            for (let i = 0; i < match_kyous_list.value.length; i++) {
                const column_query = querys.value[i]
                const target_list = match_kyous_list.value[i]
                if (!column_query || !target_list) {
                    continue
                }
                // replace は渡さない = refresh_kyou_in_list の既定の in-place splice に任せる。
                //
                // 以前は replace で `current_list.map(...)` の結果を
                // match_kyous_list.value[index] へ代入していた。これは
                //  (1) 1行の更新のために30万件の配列を2回作り直し
                //      (refresh_kyou_in_list 側の `[...list]` と、このmapの分)、
                //  (2) focused_kyous_list(= match_kyous_list[focused_column_index] への
                //      エイリアス)を黙って切る。切れると件数カレンダーとDnoteが
                //      フォーカス列に追随しなくなる。
                // in-place なら参照が保たれるので、CLAUDE.mdの局所挿入節と同じ約束を守れる。
                //
                // 「await中に列が差し替わる」ケースも in-place のほうが素直に正しい。
                // 列が再検索されていれば match_kyous_list[i] は別の配列になっており、
                // ここで掴んでいる target_list は誰も見ていないので書いても無害。
                // 書き戻す位置は refresh_kyou_in_list が await のあとにidで取り直す。
                await refresh_kyou_in_list(target_list, kyou, {
                    requested_at: requested_at,
                    query: column_query,
                })
            }
        })();
        (async (): Promise<void> => {
            if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
                const refreshed = await refresh_kyou(kyou, find_column_query_for(kyou.id), requested_at)
                if (refreshed) {
                    focused_kyou.value = refreshed
                }
            }
        })();
        (async (): Promise<void> => {
            const target_dialogs = opened_dialogs.value.filter(dialog => dialog.kyou.id === kyou.id)
            for (const target_dialog of target_dialogs) {
                const refreshed = await refresh_kyou(kyou, find_column_query_for(kyou.id), requested_at)
                if (!refreshed) {
                    continue
                }
                // await中にダイアログの開閉で並びが変わりうるので、書き戻し先はIDで引き直す。
                // 位置のまま書くと別のダイアログのkyouを差し替えてしまう
                const current_index = opened_dialogs.value.findIndex(dialog => dialog.id === target_dialog.id)
                if (current_index === -1) {
                    continue
                }
                opened_dialogs.value[current_index] = { ...opened_dialogs.value[current_index], kyou: refreshed }
            }
        })();
    }

    async function update_check_kyous(_kyou: Array<Kyou>, _is_checked: boolean): Promise<void> {
        throw new Error('Not implemented')
    }

    async function reload_list(column_index: number): Promise<void> {
        return search(column_index, querys.value[column_index], true, false, true)
    }

    async function reload_list_by_query_id(query_id: string): Promise<void> {
        const column_index = querys.value.findIndex(query => query.query_id === query_id)
        if (column_index === -1) {
            return
        }
        return reload_list(column_index)
    }

    // 追加されたKyouは再検索せず、各列の正しい位置へ差し込む。
    // 再検索するとヒット集合もスクロール位置も変わるし、KyouListViewは
    // 配列参照の差し替えでフル再描画する(reload_kyou と同じ理由)
    // onRegisteredKyou は使わない。中継ハンドラ側で requested_at を採番して
    // insert_registered_kyou を直接呼び、同じ値を変更通知にも載せる
    const { insert_registered_kyou } = useRegisteredKyouLocalInsert({
        querys: querys,
        match_kyous_list: match_kyous_list,
        reload_list_by_query_id: reload_list_by_query_id,
        onColumnMutated: (query_id: string) => {
            const column_index = querys.value.findIndex(query => query.query_id === query_id)
            if (column_index === -1 || column_index !== focused_column_index.value) {
                return
            }
            if (is_show_kyou_count_calendar.value) {
                update_focused_kyous_list(column_index)
            }
        },
    })

    // 利用者がその場で作った新しいタグを、開いている列の検索条件へ足す。
    // 既定クエリは「絞らない」を tags の列挙として物質化するので、
    // タグが1つも無い時期に作られた列は tags = ["no tags"] で凍り、
    // **タグを付けて追加した記録が追加直後に一覧から消える**（エラーも警告も出ない）
    const { onRegisteredTag: note_registered_tag, apply_new_tag_names } = useRegisteredTagColumnFilter({
        querys: querys,
        querys_backup: querys_backup,
        is_known_tag_name: (tag_name: string) => tag_exists_in_tag_struct(tag_name, props.application_config.tag_struct),
        reload_list_by_query_id: reload_list_by_query_id,
        run_with_sidebar_search_suppressed: run_with_sidebar_search_suppressed,
    })

    async function init(): Promise<void> {
        if (inited.value || is_restoring_columns.value) {
            return
        }
        // 再入ガードは同期で立てる。nextTickの中で立てると、その1tickの間に
        // もう一度呼ばれたときに二重初期化できてしまう
        is_restoring_columns.value = true
        return nextTick(async () => {
            // 起動条件がサイドバー自身の @inited ではなくなったので、テンプレートrefが
            // 埋まっていることを型の上では当てにできない。既定クエリはApplicationConfig
            // 由来なので、空のFindKyouQueryへフォールバックすると既定期間も強制非表示
            // タグも落ちた列ができる(しかもエラーは出ない)。列を作らずに戻し、
            // 次のApplicationConfig再取得でやり直せるようにする
            const sidebar = query_editor_sidebar.value
            if (!sidebar) {
                is_restoring_columns.value = false
                received_init_request.value = false
                return
            }
            const wait_promises = new Array<Promise<void>>()
            try {
                // スクロール位置の復元
                match_kyous_list_top_list.value = props.gkill_api.get_saved_mi_scroll_indexs(props.column_state_instance_key)

                // 前回開いていた列があれば復元する
                const saved_querys = props.gkill_api.get_saved_mi_find_kyou_querys(props.column_state_instance_key)
                if (saved_querys.length.valueOf() === 0) {
                    const default_query = sidebar.get_default_query()!.clone()
                    default_query.query_id = props.gkill_api.generate_uuid()
                    saved_querys.push(default_query)
                }

                // 列の骨組みを先に確定させる。hot reloadのON/OFFで分けずここで作る。
                // 検索を投げながら1本ずつ足すと「列が確定した瞬間」が定義できず、
                // 復元中にユーザが列を足したとき search(i, ...) の固定indexと衝突する。
                // querys_backup も先に埋める(サイドバーの機械的な残響が
                // search() の deep_equals 早期returnで落ちる)
                querys.value = saved_querys.concat()
                querys_backup.value = saved_querys.map(query => query.clone())
                match_kyous_list.value = saved_querys.map(() => new Array<Kyou>())

                // focused_queryの差し替えはサイドバーのprops同期を誘発するので抑止で包む。
                // 初期化の間じゅう skip_search_this_tick を立てっぱなしにしてはいけない
                // (機械的なemitが1つ届いただけで onSidebarUpdatedQuery が消費してしまい、
                //  抑止が途中で解ける)。抑止はこのコールバック式だけを使う
                run_with_sidebar_search_suppressed(() => {
                    focused_column_index.value = 0
                    focused_query.value = querys.value[0]
                })

                // ここで画面を見せる。初期検索の完了は待たない。
                // 待つと検索が1本でも解決しないだけで画面全体が固まる。
                // 進行は列ごとのスピナーとフッタの「取得中」で見せる
                // (drawer_mode_is_mobile は app_content_width の computed。ここでは触らない)
                drawer.value = props.app_content_width.valueOf() >= 760
                inited.value = true
                is_loading.value = false

                if (props.application_config.rykv_hot_reload) {
                    for (let i = 0; i < saved_querys.length; i++) {
                        await nextTick(() => {
                            // 復元は「位置を保つ再検索」。preserve_scrollを落とすと
                            // search()がscroll_to(0)を撃って保存済みの復元先を潰す
                            wait_promises.push(search(i, saved_querys[i], true, false, true).then(async () => {
                                return nextTick(() => {
                                    // 検索がエラーで早期returnしたときの保険。
                                    // 正常系ではsearch()側が同じ位置へ戻している
                                    const kyou_list_view = get_kyou_list_view(saved_querys[i].query_id)
                                    kyou_list_view?.scroll_to(match_kyous_list_top_list.value[i] ?? 0)
                                    kyou_list_view?.set_loading(false)
                                })
                            }))
                        })
                    }
                }
            } finally {
                Promise.all(wait_promises).then(() => {
                    is_restoring_columns.value = false
                })
            }
        })
    }

    async function search(column_index: number, query: FindKyouQuery, force_search?: boolean, update_cache?: boolean, preserve_scroll?: boolean): Promise<void> {
        const query_id = query.query_id
        // この列の最新の検索だけが結果を書き戻せるようにする世代番号。
        // 検索中に同じ列で条件が変更されたら、最後の検索条件の結果だけを表示する
        let seq = -1
        const is_current = () => seq !== -1 && search_seqs.get(query_id) === seq

        // 検索する。Tickでまとめる
        try {
            if (!force_search) {
                if (deep_equals(querys_backup.value[column_index], query)) {
                    return
                }
            }

            seq = (search_seqs.get(query_id) ?? 0) + 1
            search_seqs.set(query_id, seq)
            // 採番できた回だけ数える。deep_equalsの早期returnはseq=-1のままなので数えない
            running_search_count.value++

            querys.value[column_index] = query
            // バックアップは複製で持つ。同一参照だと後からの変更が両方に映って差分検知が壊れる
            querys_backup.value[column_index] = query.clone()
            // フォーカス列の検索だけがサイドバーの表示対象(focused_query)を更新する。
            // 無条件に更新すると全列リロード等でサイドバーが別列の条件に乗っ取られ、
            // そのquery_idを引き継いだ編集が列間のquery_id重複(結果の誤配送)を生む。
            // 差し替えはサイドバーのprops同期を誘発するので、機械的な残響を1回分抑止する
            if (column_index === focused_column_index.value) {
                run_with_sidebar_search_suppressed(() => {
                    focused_query.value = query
                })
            }

            props.gkill_api.set_saved_mi_find_kyou_querys(querys.value, props.column_state_instance_key)

            // 前の検索処理を中断する
            abort_controllers.get(query_id)?.abort()

            if (match_kyous_list.value[column_index]) {
                match_kyous_list.value[column_index] = []
            }

            nextTick(() => {
                if (!is_current()) {
                    return
                }
                const kyou_list_view = get_kyou_list_view(query_id)
                if (kyou_list_view) {
                    if (inited.value && !preserve_scroll) {
                        kyou_list_view.scroll_to(0)
                    }
                    ((async () => kyou_list_view.set_loading(true))());
                }
            })

            const wait_promises = new Array<Promise<unknown>>()

            const req = new GetKyousRequest()
            abort_controllers.set(query_id, req.abort_controller)
            req.query = query.clone()
            req.query.parse_words_and_not_words()
            if (update_cache) {
                wait_promises.push(delete_gkill_kyou_cache(null))
                req.query.update_cache = true
            } else {
                wait_promises.push(props.gkill_api.delete_updated_gkill_caches())
            }

            let res = new GetKyousResponse()
            wait_promises.push(props.gkill_api.get_kyous(req).then(response => res = response))

            await Promise.all(wait_promises)

            // 待っている間に同じ列で新しい検索が始まっていたら、この結果は捨てる。
            // スピナーは新しい検索が所有しているので触らない
            if (!is_current()) {
                return
            }

            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                // エラー時もスピナーを解除する。解除しないと列が読み込み中のまま残る
                nextTick(() => get_kyou_list_view(query_id)?.set_loading(false))
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }

            // 検索後の列位置を取得する
            column_index = -1
            for (let i = 0; i < querys.value.length; i++) {
                const query = querys.value[i]
                if (query.query_id === query_id) {
                    column_index = i
                    break
                }
            }

            if (column_index === -1) {
                return
            }

            match_kyous_list.value[column_index] = res.kyous
            // フォーカス列以外の検索完了でfocused_kyous_list(カレンダー)を汚染しない
            if (column_index === focused_column_index.value && is_show_kyou_count_calendar.value) {
                update_focused_kyous_list(column_index)
            }

            await nextTick(() => {
                if (!is_current()) {
                    return
                }
                const kyou_list_view = get_kyou_list_view(query_id)
                if (kyou_list_view) {
                    ((async () => kyou_list_view.set_loading(false))());
                    if (inited.value) {
                        if (preserve_scroll) {
                            kyou_list_view.scroll_to(match_kyous_list_top_list.value[column_index] ?? 0)
                        } else {
                            kyou_list_view.scroll_to(0)
                        }
                    }
                }
                // 抑止を倒すのは run_with_sidebar_search_suppressed の1tickが本筋。
                // ここは「検索が終わったのに抑止が残っている」を掃除する保険
                skip_search_this_tick.value = false
            })
        } catch (err: unknown) {
            // 中断（画面を離れた・後発の検索に差し替わった）は正常なので出さない
            log_unless_aborted(err)
            // abort含め例外時はloading状態を解除する（ただし新しいsearchが開始されていない場合のみ）
            if (is_current()) {
                nextTick(() => get_kyou_list_view(query_id)?.set_loading(false))
            }
        } finally {
            // 採番できた回だけ減らす。増やしたのと同じ条件にする
            if (seq !== -1) {
                running_search_count.value--
            }
        }
    }

    async function close_list_view(column_index: number): Promise<void> {
        return nextTick(() => {
            // ここでの書き込み(focused_column_index・列配列のsplice)はサイドバーの
            // props(focused_query)に触れないので抑止は不要。抑止が要るのは
            // focused_queryを差し替える下のnextTick内だけ
            const closed_query = querys.value[column_index]
            const focused_index_before_close = focused_column_index.value
            focused_column_index.value = -1

            querys.value.splice(column_index, 1)
            querys_backup.value.splice(column_index, 1)

            if (closed_query) {
                // 飛行中の検索は中断し、世代表からも消す。遅れて届いた結果は世代照合で破棄される
                abort_controllers.get(closed_query.query_id)?.abort()
                abort_controllers.delete(closed_query.query_id)
                search_seqs.delete(closed_query.query_id)
            }

            match_kyous_list.value.splice(column_index, 1)
            match_kyous_list_top_list.value.splice(column_index, 1)

            props.gkill_api.set_saved_mi_find_kyou_querys(querys.value, props.column_state_instance_key)
            props.gkill_api.set_saved_mi_scroll_indexs(match_kyous_list_top_list.value, props.column_state_instance_key)
            nextTick(() => {
                // 閉じた列の最近傍の列へフォーカスを移す
                const next_focused_index = Math.max(0, Math.min(
                    focused_index_before_close > column_index ? focused_index_before_close - 1 : focused_index_before_close,
                    querys.value.length - 1,
                ))
                run_with_sidebar_search_suppressed(() => {
                    focused_column_index.value = next_focused_index
                    focused_query.value = querys.value[next_focused_index]
                })
                if (is_show_kyou_count_calendar.value) {
                    update_focused_kyous_list(next_focused_index)
                }
            })
        })
    }

    function add_list_view(query?: FindKyouQuery): void {
        match_kyous_list.value.push(new Array<Kyou>())
        match_kyous_list_top_list.value.push(0)
        // 初期化されていないときはDefaultQueryがない。
        // その場合は初期値のFindKyouQueryをわたして初期化してもらう
        const default_query = query_editor_sidebar.value?.get_default_query()?.clone()
        if (query) {
            querys.value.push(query)
            focused_query.value = query
        } else if (default_query) {
            default_query.query_id = props.gkill_api.generate_uuid()
            querys.value.push(default_query)
            focused_query.value = default_query
        } else {
            const query = new FindKyouQuery()
            query.query_id = props.gkill_api.generate_uuid()
            querys.value.push(query)
            focused_query.value = query
        }
        if (inited.value) {
            focused_column_index.value = querys.value.length - 1
        }
        props.gkill_api.set_saved_mi_find_kyou_querys(querys.value, props.column_state_instance_key)
        props.gkill_api.set_saved_mi_scroll_indexs(match_kyous_list_top_list.value, props.column_state_instance_key)
    }

    function open_or_focus_board(board_name: string): void {
        // 「すべて」の番兵はサイドバーのgenerate_queryが比較に使うキーと揃える。
        // ロケール非依存の MI_ALL_BOARD_KEY（＝ツリーのノードが持つkey）であること
        const all_board_sentinel = MI_ALL_BOARD_KEY
        if (board_name === "") {
            board_name = all_board_sentinel
        }
        const is_all_board = board_name === all_board_sentinel

        let opened = false
        for (let i = 0; i < querys.value.length; i++) {
            const query = querys.value[i]
            // 「すべて」列は mi_board_name === null で表す。
            // 板名の文字列一致だけだと null(すべて)列を拾えないので null 判定で分ける
            const is_match = is_all_board
                ? query.mi_board_name === null
                : (query.mi_board_name !== null && query.mi_board_name === board_name)
            if (is_match) {
                // フォーカス切り替えの他経路と同様、機械的なサイドバー更新を1回分抑止する
                run_with_sidebar_search_suppressed(() => {
                    focused_query.value = querys.value[i]
                    focused_kyous_list.value = match_kyous_list.value[i] ?? []
                    focused_column_index.value = i
                })
                opened = true
                break
            }
        }
        if (opened) {
            return
        }

        const query = query_editor_sidebar.value!.get_default_query()!.clone()
        query.query_id = props.gkill_api.generate_uuid()
        query.mi_board_name = is_all_board ? null : board_name

        // add_list_viewはfocused_queryを差し替えるので抑止で包む
        run_with_sidebar_search_suppressed(() => {
            add_list_view(query)
        })
        if (props.application_config.rykv_hot_reload) {
            search(querys.value.length - 1, query, true)
        }
    }

    // ダイアログの中でフォーカスが動いたときに列を追随させる。
    // rykv の同名の束（use-rykv-view.ts の subViewFocusHandlers）と対。
    // 列のクリック（clicked_kyou_in_list_view）と違って列番号が無いので、
    // フォーカス列の付け替えはしない
    function onFocusedKyouFromSubView(kyou: Kyou): void {
        focused_kyou.value = kyou
        if (!inited.value || !kyou_list_views.value) {
            return
        }
        for (let i = 0; i < querys.value.length; i++) {
            if (querys.value[i].is_focus_kyou_in_list_view) {
                get_kyou_list_view(querys.value[i].query_id)?.scroll_to_time(kyou.related_time)
            }
        }
    }

    const subViewFocusHandlers = {
        'focused_kyou': (kyou: Kyou) => onFocusedKyouFromSubView(kyou),
        'clicked_kyou': (kyou: Kyou) => onFocusedKyouFromSubView(kyou),
    }

    // 列からの「一覧を引き直して」。クエリを clone してから引き直すのは、
    // 検索の同値ガード（deep_equals）に「変わった」と認識させるため。
    // rykv の同名の関数と対
    function onColumnRequestedReloadList(index: number): void {
        const query = querys.value[index].clone()
        querys.value[index] = query
        reload_list(index)
    }

    async function clicked_kyou_in_list_view(column_index: number, kyou: Kyou): Promise<void> {
        focused_kyou.value = kyou
        focused_column_index.value = column_index

        const update_target_column_indexs = new Array<number>()
        for (let i = 0; i < querys.value.length; i++) {
            if (querys.value[i].is_focus_kyou_in_list_view) {
                update_target_column_indexs.push(i)
            }
        }

        for (let i = 0; i < update_target_column_indexs.length; i++) {
            const target_column_index = update_target_column_indexs[i]
            if (inited.value) {
                get_kyou_list_view(querys.value[target_column_index].query_id)?.scroll_to_time(kyou.related_time)
            }
        }
    }

    // ドロップは板の移動をそのままサーバへ書き込む。前の移動が終わる前に
    // 次をドロップされると、同じタスクに対する更新が重なるので直列化する
    async function onDropBoardTask(e: DragEvent, find_kyou_query: FindKyouQuery) {
        if (is_moving_board_task.value) {
            return
        }
        is_moving_board_task.value = true
        try {
            await move_board_task(e, find_kyou_query)
        } finally {
            is_moving_board_task.value = false
        }
    }

    async function move_board_task(e: DragEvent, find_kyou_query: FindKyouQuery) {
        // MiとMiReKyouのどちらがドロップされたか判定する
        const mi_json = e.dataTransfer!.getData("gkill_mi")
        const mirekyou_json = e.dataTransfer!.getData("gkill_mi_re_kyou")
        let task: Mi | MiReKyou
        try {
            if (mirekyou_json !== "") {
                task = parse_dropped_task(JSON.parse(mirekyou_json), new MiReKyou(), MI_REKYOU_DROP_ALLOWED_KEYS)
            } else if (mi_json !== "") {
                task = parse_dropped_task(JSON.parse(mi_json), new Mi(), MI_DROP_ALLOWED_KEYS)
            } else {
                return
            }
        } catch (e: unknown) {
            console.error(e)
            return
        }

        if (!task.id || task.id == "") {
            return
        }

        e!.preventDefault()
        e!.stopPropagation()

        const before_board_name = task.board_name
        const after_board_name = find_kyou_query.mi_board_name
        // 「すべて」列(mi_board_name === null)へのドロップは移動先の板が定まらないので何もしない
        if (after_board_name === null || before_board_name === after_board_name) {
            return
        }

        // 新しい板名の確認はここでは行わない。移動先の列は板ツリー由来なので、
        // ツリーを正とする判定では必ず既存扱いになり、確認が発火しないため
        task.board_name = after_board_name
        task.update_app = "gkill"
        task.update_device = props.application_config.device
        task.update_time = new Date(Date.now())
        task.update_user = props.application_config.user_id

        let updated_task: Mi | MiReKyou = task
        let updated_kyou: Kyou | null = null
        if (task instanceof MiReKyou) {
            const req = new UpdateMiReKyouRequest()
            req.mirekyou = task
            req.want_response_kyou = true
            const res = await props.gkill_api.update_mirekyou(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            if (res.updated_mirekyou && res.updated_mirekyou.id !== "") {
                updated_task = res.updated_mirekyou
            }
            updated_kyou = res.updated_kyou
        } else {
            const req = new UpdateMiRequest()
            req.mi = task
            req.want_response_kyou = true
            const res = await props.gkill_api.update_mi(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            if (res.updated_mi && res.updated_mi.id !== "") {
                updated_task = res.updated_mi
            }
            updated_kyou = res.updated_kyou
        }

        apply_board_move_locally(task.id, before_board_name, after_board_name, updated_task)

        if (updated_kyou) {
            emits('updated_kyou', updated_kyou)
        }
    }

    function onDragoverBoardTask(e: DragEvent, _find_kyou_query: FindKyouQuery) {
        e!.dataTransfer!.dropEffect = "move"
        e!.preventDefault()
        e!.stopPropagation()
    }

    function open_rykv_dialog(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload): void {
        const dialog_id = props.gkill_api.generate_uuid()
        opened_dialogs.value.push({
            id: dialog_id,
            kind,
            kyou: kyou.clone(),
            payload: payload ?? null,
            opened_at: Date.now(),
        })
        // 開いた直後にも最新化する。リストのKyouは検索時点のものなので、
        // 別経路で更新されていると古い内容でダイアログが開いてしまう
        ;(async (): Promise<void> => {
            const refreshed = await refresh_kyou(kyou, find_column_query_for(kyou.id))
            if (!refreshed) {
                return
            }
            for (let i = 0; i < opened_dialogs.value.length; i++) {
                if (opened_dialogs.value[i].id === dialog_id) {
                    opened_dialogs.value[i] = { ...opened_dialogs.value[i], kyou: refreshed }
                    return
                }
            }
        })().catch((err: unknown) => console.error(err))
    }

    function close_rykv_dialog(dialog_id: string): void {
        for (let i = 0; i < opened_dialogs.value.length; i++) {
            if (opened_dialogs.value[i].id === dialog_id) {
                opened_dialogs.value.splice(i, 1)
                break
            }
        }
    }

    // ── Template event handlers (extracted from inline) ──

    function toggle_drawer(): void {
        if (inited.value) { drawer.value = !drawer.value }
    }

    async function navigate_to_page(page_name: string): Promise<void> {
        // ポート(rudbeckia)の中では行き先を親が決める。
        // ここで reset_dialog_history() を呼んではいけない ―― モジュール共有の履歴スタックを
        // 巻き戻すので、ポートで開いている他のウィンドウまで一斉に閉じる
        if (props.is_hosted_in_dialog) {
            emits('requested_navigate_page', page_name)
            return
        }
        await reset_dialog_history()
        router.replace('/' + page_name + '?loaded=true')
    }

    function onSidebarRequestedSearch(update_cache: boolean): void {
        const column_index = focused_column_index.value
        const base_query = querys.value[column_index]
        if (!base_query) {
            return
        }
        // 検索ボタンはサイドバーに今見えている条件で検索する。
        // hot reload OFFでは編集が列のクエリに保存されていないため、保存済みクエリではなく
        // サイドバーから現在値を組み立てる(query_idは列のものを引き継ぐ)
        const current_query = query_editor_sidebar.value?.generate_query(base_query.query_id) ?? base_query
        nextTick(() => search(column_index, current_query, true, update_cache))
    }

    function onSidebarUpdatedQuery(new_query: FindKyouQuery): void {
        // 「初期化が終わるまで捨てる」ガードは置かない。初期検索の飛行中でも
        // ユーザの編集は通してよく、むしろ通すのが正しい。復元検索とユーザ検索は
        // 同じ query_id を共有するので、search() の abort_controllers が復元を中断し、
        // search_seqs の世代照合が遅れて届いた復元結果を捨てる(=ユーザが勝つ)。
        // サイドバーの機械的な残響は emits_current_query の値比較ガードと
        // search() の deep_equals 早期returnで落ちる
        if (skip_search_this_tick.value || !props.application_config.rykv_hot_reload) {
            nextTick(() => skip_search_this_tick.value = false)
            return
        }
        // どの列宛ての編集かはquery_idで解決する。focused_column_indexを盲信すると、
        // フォーカス切り替えと編集が交錯したときに別列へ書き込んでしまう
        const column_index = querys.value.findIndex(q => q.query_id === new_query.query_id)
        if (column_index === -1) {
            return
        }
        search(column_index, new_query)
    }

    function onColumnScrollList(index: number, scroll_top: number): void {
        // 検索中の列のスクロール通知は、リストを空にした副作用で届く機械的なもの。
        // 取り込むと preserve_scroll の復元先が0で潰れ、そのまま保存位置にも焼き付く
        const column_query = querys.value[index]
        if (column_query && get_kyou_list_view(column_query.query_id)?.get_is_loading()) {
            return
        }
        match_kyous_list_top_list.value[index] = scroll_top
        props.gkill_api.set_saved_mi_scroll_indexs(match_kyous_list_top_list.value, props.column_state_instance_key)
    }

    function onColumnClickedListView(index: number): void {
        run_with_sidebar_search_suppressed(() => {
            focused_column_index.value = index
            focused_query.value = querys.value[index]
        })
        if (is_show_kyou_count_calendar.value) {
            update_focused_kyous_list(index)
        }
    }

    function onColumnClickedKyou(index: number, kyou: Kyou): void {
        run_with_sidebar_search_suppressed(() => {
            focused_column_index.value = index
            focused_query.value = querys.value[index]
        })
        clicked_kyou_in_list_view(index, kyou)
    }

    function onColumnRequestedChangeFocusKyou(index: number, is_focus: boolean): void {
        run_with_sidebar_search_suppressed(() => {
            focused_column_index.value = index
            const query = querys.value[index].clone()
            query.is_focus_kyou_in_list_view = is_focus
            querys.value.splice(index, 1, query)
            querys_backup.value.splice(index, 1, query.clone())
            // サイドバーの表示対象も差し替え後のクエリへ揃える
            focused_query.value = query
        })
    }

    function onColumnRequestedSearch(index: number): void {
        run_with_sidebar_search_suppressed(() => {
            focused_column_index.value = index
            const query = querys.value[index].clone()
            querys.value.splice(index, 1, query)
            querys_backup.value.splice(index, 1, query.clone())
        })
        reload_list(index)
    }

    function onColumnRequestedChangeImageOnlyView(index: number, is_image_only: boolean): void {
        run_with_sidebar_search_suppressed(() => {
            focused_column_index.value = index
            focused_kyous_list.value = match_kyous_list.value[index]
            const query = querys.value[index].clone()
            query.is_image_only = is_image_only
            querys.value.splice(index, 1, query)
            querys_backup.value.splice(index, 1, query.clone())
        })
        reload_list(index)
    }

    function onRequestedFocusTime(time: Date): void {
        focused_time.value = time
    }

    // ── Dialog show methods ──
    function show_kftl_dialog(): void {
        kftl_dialog.value?.show()
    }

    function show_add_kc_dialog(): void {
        add_kc_dialog.value?.show()
    }

    function show_mkfl_dialog(): void {
        mkfl_dialog.value?.show()
    }

    function show_timeis_dialog(): void {
        add_timeis_dialog.value?.show()
    }

    function show_mi_dialog(): void {
        add_mi_dialog.value?.show()
    }

    function show_nlog_dialog(): void {
        add_nlog_dialog.value?.show()
    }

    function show_lantana_dialog(): void {
        add_lantana_dialog.value?.show()
    }

    function show_urlog_dialog(): void {
        add_urlog_dialog.value?.show()
    }

    function show_upload_file_dialog(): void {
        upload_file_dialog.value?.show()
    }

    function show_save_clipboard_to_file_dialog(): void {
        save_clipboard_to_file_dialog.value?.show()
    }

    function floating_action_button_style() {
        return {
            'bottom': '60px',
            'right': '10px',
            'height': '50px',
            'width': '50px'
        }
    }

    // ── Event relay objects ──
    const crudRelayHandlers = {
        'deleted_kyou': (kyou: Kyou) => { onDeletedKyou(kyou); publish_kyou_change({ kind: 'deleted', kyou: kyou }, new_reload_batch()) },
        'deleted_tag': (tag: Tag) => emits('deleted_tag', tag),
        'deleted_text': (text: Text) => emits('deleted_text', text),
        'deleted_notification': (notification: Notification) => emits('deleted_notification', notification),
        'registered_kyou': (kyou: Kyou) => {
            // 引き直しの合流キーはここで採番する。ホスト側で採番すると
            // kyou-reload.ts の合流条件に間に合わず、画面の枚数ぶん往復する
            const requested_at = new_reload_batch()
            void insert_registered_kyou(kyou, requested_at)
            emits('registered_kyou', kyou)
            publish_kyou_change({ kind: 'registered', kyou: kyou }, requested_at)
        },
        'registered_tag': (tag: Tag) => {
            // **「未知だったか」は emit より前に、同期で決めること。**
            // emit 先(use-mi-page.ts)の check_tag_update がタグツリーへ足したあとでは、
            // 「利用者がついさっき作った」ことを二度と知れない
            if (note_registered_tag(tag)) {
                // 未知と判定した発生元だけが配る。受け手は判定をやり直さない
                publish_kyou_change({ kind: 'registered_tag', tag_name: tag.tag }, new_reload_batch())
            }
            emits('registered_tag', tag)
        },
        'registered_text': (text: Text) => emits('registered_text', text),
        'registered_notification': (notification: Notification) => emits('registered_notification', notification),
        'updated_kyou': (kyou: Kyou) => {
            const requested_at = new_reload_batch()
            void reload_kyou(kyou, requested_at)
            emits('updated_kyou', kyou)
            publish_kyou_change({ kind: 'reload', kyou: kyou }, requested_at)
        },
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'updated_text': (text: Text) => emits('updated_text', text),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    const allColumnsRequestHandlers = {
        'requested_reload_kyou': (kyou: Kyou) => {
            const requested_at = new_reload_batch()
            void reload_kyou(kyou, requested_at)
            publish_kyou_change({ kind: 'reload', kyou: kyou }, requested_at)
        },
        'requested_reload_list': () => {
            for (let i = 0; i < querys.value.length; i++) {
                void reload_list(i)
            }
            publish_kyou_change({ kind: 'reload_list' }, new_reload_batch())
        },
        'requested_update_check_kyous': (kyous: Array<Kyou>, checked: boolean) => update_check_kyous(kyous, checked),
    }

    const rykvDialogRelayHandlers = {
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => open_rykv_dialog(kind, kyou, payload),
    }


    // ── 画面間の変更通知 ──
    // ポート(rudbeckia)で複数の画面を並べているとき、ここで起きた変更を他の画面へ配る。
    // チャネルが null（単独ページ）なら何も起きない
    function publish_kyou_change(change: KyouChange, requested_at: number): void {
        const channel = props.kyou_change_channel
        if (!channel) {
            return
        }
        channel.bus.publish(channel.origin_id, change, requested_at)
    }
    // 他の画面で起きた変更を反映する。**適用関数だけを渡すこと** ――
    // 中継束を渡すと emit が走ってホストが再 publish し、通知が無限に往復する
    useKyouChangeSubscriber(() => props.kyou_change_channel, {
        apply_registered: (kyou, requested_at) => { void insert_registered_kyou(kyou, requested_at) },
        apply_reload: (kyou, requested_at) => { void reload_kyou(kyou, requested_at) },
        apply_deleted: (kyou) => apply_deleted_kyou(kyou),
        apply_reload_list: () => {
            for (let i = 0; i < querys.value.length; i++) {
                void reload_list(i)
            }
        },
        // 他の画面で新しく作られたタグ。**既知判定はやり直さない**
        // （届く頃には発生元の check_tag_update がツリーへ足し終えている）
        apply_registered_tag: (tag_names) => apply_new_tag_names(tag_names),
    })

    // ── Keyboard shortcut ──
    // window にキャプチャで張るので、ポート(rudbeckia)で複数枚ホストすると多重登録になる。
    // 意味は use-rykv-view.ts の同じ箇所と同じ（対称実装）
    const enable_enter_shortcut = computed(() => !props.is_hosted_in_dialog)
    useScopedEnterForKFTL(mi_root, show_kftl_dialog, enable_enter_shortcut)
    useScopedCtrlVForClipboard(mi_root, show_save_clipboard_to_file_dialog, enable_enter_shortcut)

    // ── Return ──
    return {
        // Template refs
        mi_root,
        query_editor_sidebar,
        add_mi_dialog,
        add_nlog_dialog,
        add_lantana_dialog,
        add_timeis_dialog,
        add_urlog_dialog,
        kftl_dialog,
        add_kc_dialog,
        mkfl_dialog,
        upload_file_dialog,
        save_clipboard_to_file_dialog,
        kyou_list_views,

        // State
        enable_context_menu,
        enable_dialog,
        opened_dialogs,
        querys,
        match_kyous_list,
        focused_query,
        focused_column_index,
        focused_kyous_list,
        focused_kyou,
        is_show_kyou_detail_view,
        is_show_kyou_count_calendar,
        drawer,
        drawer_mode_is_mobile,
        is_loading,
        inited,
        is_restoring_columns,

        // Computed
        kyou_list_view_height,
        page_list,
        is_view_ready,

        // Template event handlers
        toggle_drawer,
        navigate_to_page,
        onSidebarRequestedSearch,
        onSidebarUpdatedQuery,
        onColumnScrollList,
        onColumnClickedListView,
        onColumnClickedKyou,
        onColumnRequestedChangeFocusKyou,
        onColumnRequestedSearch,
        onColumnRequestedChangeImageOnlyView,
        onRequestedFocusTime,
        onDropBoardTask,
        onDragoverBoardTask,
        close_list_view,
        open_or_focus_board,
        open_rykv_dialog,
        close_rykv_dialog,
        reload_kyou,
        reload_list,
        insert_registered_kyou,
        update_check_kyous,

        // Dialog show methods
        show_kftl_dialog,
        show_mkfl_dialog,
        show_add_kc_dialog,
        show_urlog_dialog,
        show_timeis_dialog,
        show_mi_dialog,
        show_nlog_dialog,
        show_lantana_dialog,
        show_upload_file_dialog,
        show_save_clipboard_to_file_dialog,
        floating_action_button_style,

        // Event relay objects
        crudRelayHandlers,
        allColumnsRequestHandlers,
        subViewFocusHandlers,
        onColumnRequestedReloadList,
        rykvDialogRelayHandlers,
    }
}
