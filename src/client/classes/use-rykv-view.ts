import { gkill_page_list } from '@/classes/gkill-page-list'
import type { KyouChange } from '@/classes/kyou-change-bus'
import { useKyouChangeSubscriber } from '@/classes/use-kyou-change-subscriber'
import router from '@/router'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { computed, nextTick, onBeforeUnmount, type Ref, ref, toRaw, watch } from 'vue'
import { Kyou } from '@/classes/datas/kyou'
import type { RykvViewEmits } from '@/pages/views/rykv-view-emits'
import type { RykvViewProps } from '@/pages/views/rykv-view-props'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import { GetKyousResponse } from '@/classes/api/req_res/get-kyous-response'
import moment from 'moment'
import { deep_equals } from '@/classes/deep-equals'
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'
import { useScopedCtrlVForClipboard } from '@/classes/use-scoped-ctrl-v-for-clipboard'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { Tag } from '@/classes/datas/tag'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import { build_mi_reload_query, new_reload_batch, refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import { useRegisteredKyouLocalInsert } from '@/classes/use-registered-kyou-local-insert'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { ComponentRef } from '@/classes/component-ref'

export function useRykvView(options: {
    props: RykvViewProps,
    emits: RykvViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const rykv_root = ref<HTMLElement | null>(null)
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
    const dnote_view = ref<ComponentRef | null>(null)
    const kyou_list_views = ref()
    const kyou_detail_view_element = ref<HTMLElement | null>(null)

    // ── State refs ──
    const enable_context_menu = ref(true)
    const enable_dialog = ref(true)
    const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])

    const querys: Ref<Array<FindKyouQuery>> = ref([new FindKyouQuery()])
    const querys_backup: Ref<Array<FindKyouQuery>> = ref(new Array<FindKyouQuery>()) // 更新検知用バックアップ
    const match_kyous_list: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
    const match_kyous_list_top_list: Ref<Array<number>> = ref(new Array<number>())
    const focused_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const focused_column_index: Ref<number> = ref(0)
    const focused_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const focused_kyou: Ref<Kyou | null> = ref(null)
    const focused_time: Ref<Date> = ref(moment().toDate())
    const focused_column_checked_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const gps_log_map_start_time: Ref<Date> = ref(moment().toDate())
    const gps_log_map_end_time: Ref<Date> = ref(moment().toDate())
    const gps_log_map_marker_time: Ref<Date> = ref(moment().toDate())
    const is_show_kyou_detail_view: Ref<boolean> = ref(false)
    const is_show_kyou_count_calendar: Ref<boolean> = ref(false)
    const is_show_gps_log_map: Ref<boolean> = ref(false)
    const is_show_dnote: Ref<boolean> = ref(false)
    const drawer: Ref<boolean | null> = ref(false)
    // 一時表示(オーバーレイ)モードにするかは「今の内容領域の幅」で決まる。
    // 初期化時に1回代入するだけにしてはいけない ―― このビューはポート(rudbeckia)の
    // リサイズできるダイアログの中でも描かれるので、幅はあとから変わる
    const drawer_mode_is_mobile = computed<boolean>(() => !(props.app_content_width.valueOf() >= 760))
    const default_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const is_loading: Ref<boolean> = ref(true)
    const inited = ref(false)
    const is_restoring_columns = ref(false) // 保存済み列の初期検索がまだ走っている。表示制御には使わない
    const running_search_count = ref(0) // 飛行中の検索の本数。準備完了信号にだけ使う
    const received_init_request = ref(false)
    const skip_search_this_tick = ref(false)
    const abort_controllers = new Map<string, AbortController>() // 列ごとの検索中断用。キーは列のquery_id
    const search_seqs = new Map<string, number>() // 列ごとの検索の世代番号。キーは列のquery_id。最後の検索だけが結果を書き戻せるようにする
    const dnote_reload_seq = ref(0) // Dnote再集計の世代番号。列を連打したとき古い再集計が後勝ちしないようにする
    const kyou_detail_view_width: Ref<number> = ref(400) // KyouDetailViewの初期幅とあわせる。ryuuの最大幅に使う

    // ── Computed ──
    const kyou_list_view_height = computed(() => props.app_content_height)

    // 「操作してよい状態」をE2Eが決定論的に待つための信号。
    // 画面の表示/非表示には一切使わない(使うと初期検索を待たなくした意味がなくなる)
    const is_view_ready = computed(() =>
        inited.value && !is_restoring_columns.value && running_search_count.value === 0)

    // 画面切替メニューの一覧は classes/gkill-page-list.ts に1つだけ置いてある
    const page_list = gkill_page_list

    // ── Watchers ──
    // KyouDetailViewはCSSのresizeでユーザが幅を変えるため、実寸をResizeObserverで追う。
    // 表示切り替えでelementごと作り直されるので、refをwatchして付け替える。
    let kyou_detail_view_resize_observer: ResizeObserver | null = null
    watch(kyou_detail_view_element, (element, old_element) => {
        if (kyou_detail_view_resize_observer && old_element) {
            try { kyou_detail_view_resize_observer.unobserve(old_element) } catch { /* noop */ }
        }
        if (element) {
            if (!kyou_detail_view_resize_observer) {
                kyou_detail_view_resize_observer = new ResizeObserver((entries) => {
                    for (const entry of entries) {
                        kyou_detail_view_width.value = entry.borderBoxSize?.[0]?.inlineSize ?? entry.contentRect.width
                    }
                })
            }
            kyou_detail_view_resize_observer.observe(element)
        }
    }, { flush: 'post' })
    onBeforeUnmount(() => {
        kyou_detail_view_resize_observer?.disconnect()
        kyou_detail_view_resize_observer = null
        // 飛行中の検索を止める。ポート(rudbeckia)ではこのビューごと閉じられるので、
        // 放っておくと閉じたウィンドウぶんの数十万件の取得が最後まで走る
        for (const abort_controller of abort_controllers.values()) {
            abort_controller.abort()
        }
        abort_controllers.clear()
    })

    watch(() => focused_time.value, () => {
        // 共有ページでは初期クエリのquery_idが空文字のまま使われるので、
        // 「列が無い」判定はquery_idの真偽値ではなく列の存在で行う
        const focused_column_query = querys.value[focused_column_index.value]
        if (!focused_column_query) {
            return
        }
        const kyou_list_view = get_kyou_list_view(focused_column_query.query_id)
        if (!kyou_list_view) {
            return
        }
        if (inited.value) {
            kyou_list_view.scroll_to_time(focused_time.value)
        }
    })

    watch(() => is_show_kyou_count_calendar.value, () => {
        if (props.is_shared_rykv_view) {
            return
        }
        if (is_show_kyou_count_calendar.value) {
            update_focused_kyous_list(focused_column_index.value)
        }
    })

    watch(() => is_show_dnote.value, async () => {
        if (props.is_shared_rykv_view) {
            return
        }
        if (is_show_dnote.value) {
            await reload_dnote_for_column(focused_column_index.value)
        } else {
            // 実行中の再集計を無効化してから止める
            dnote_reload_seq.value++
            await dnote_view.value?.abort()
        }
    })

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
        if (props.is_shared_rykv_view) {
            // 共有画面は下の Shared view init が別途初期化する
            return
        }
        if (received_init_request.value) {
            return
        }
        received_init_request.value = true
        init()
    }, { immediate: true })

    // ── Shared view init ──
    if (props.is_shared_rykv_view) {
        nextTick(async () => {
            is_loading.value = false
            inited.value = true
            await props.gkill_api.delete_updated_gkill_caches()
            const kyous = (await props.gkill_api.get_kyous(new GetKyousRequest())).kyous
            const wait_promises = new Array<Promise<unknown>>()
            for (let i = 0; i < kyous.length; i++) {
                wait_promises.push(kyous[i].load_all())
            }
            await Promise.all(wait_promises)
            match_kyous_list.value = [kyous]
            focused_kyous_list.value = kyous
            focused_column_index.value = 0
        })
    }

    // ── Internal helpers ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

    // 列コンポーネントをquery_idで引く。v-forのテンプレートref配列はマウント順で
    // 並び順が保証されないため、indexではなくquery_idで解決する
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    function get_kyou_list_view(query_id: string): any {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        return kyou_list_views.value?.filter((kyou_list_view: any) => kyou_list_view.get_query_id() === query_id)[0]
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

    function update_focused_kyous_list(column_index: number): void {
        if (props.is_shared_rykv_view) {
            return
        }
        if (!match_kyous_list.value || match_kyous_list.value.length === 0) {
            return
        }
        focused_kyous_list.value = match_kyous_list.value[column_index]
    }

    // 指定列の内容でDnoteを再集計する。
    // Dnote非表示・共有画面・列が存在しない場合は何もしない
    async function reload_dnote_for_column(column_index: number): Promise<void> {
        if (props.is_shared_rykv_view) {
            return
        }
        if (!is_show_dnote.value) {
            return
        }
        const target_query = querys.value[column_index]
        if (!target_query) {
            return
        }

        const seq = ++dnote_reload_seq.value
        update_focused_kyous_list(column_index)

        // Dnoteはv-ifでマウントされるのでrefが生えるまで1tick待つ
        await nextTick()
        if (seq !== dnote_reload_seq.value) {
            return
        }
        await dnote_view.value?.abort()
        if (seq !== dnote_reload_seq.value) {
            return
        }

        // 対象列がまだ検索中なら終わるまで待つ
        const kyou_list_view = get_kyou_list_view(target_query.query_id)
        if (kyou_list_view && kyou_list_view.get_is_loading()) {
            dnote_view.value?.set_loading(true)
            while (kyou_list_view.get_is_loading()) {
                await sleep(500)
                if (seq !== dnote_reload_seq.value) {
                    return
                }
            }
            // 待っている間に検索結果が差し替わっている(列の位置も動きうる)ので取り直す
            const current_index = querys.value.findIndex(q => q.query_id === target_query.query_id)
            if (current_index === -1) {
                return
            }
            update_focused_kyous_list(current_index)
        }

        try {
            await dnote_view.value?.reload(focused_kyous_list.value, target_query)
        } catch (err: unknown) {
            // abortは握りつぶす
            if (!(err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request")))) {
                console.error(err)
            }
        }
    }

    // 列フォーカスを切り替える。列が実際に変わったときだけDnoteを再集計する
    function focus_column(index: number): void {
        const is_column_changed = focused_column_index.value !== index
        focused_column_index.value = index
        focused_query.value = querys.value[index]
        if (is_show_kyou_count_calendar.value || is_show_dnote.value) {
            update_focused_kyous_list(index)
        }
        if (is_column_changed) {
            // 同じ列内の連続クリックでは重い再集計を走らせない
            reload_dnote_for_column(index)
        }
    }

    function remove_kyou_from_list_by_id(list: Array<Kyou>, deleted_id: string): void {
        // 走査は生の配列に対して行う。deepなref配下のリアクティブProxy越しに読むと
        // 1要素ごとに track と toReactive が走り、要素ぶんのProxyを確保する(30万件の列では効く)。
        // ★splice は必ずリアクティブな list に対して行うこと(でないと誰にも通知されない)。
        //   後ろから走るので、splice しても未走査側のインデックスはずれない。
        const raw_list = toRaw(list)
        for (let i = raw_list.length - 1; i >= 0; i--) {
            if (raw_list[i].id === deleted_id) {
                list.splice(i, 1)
            }
        }
    }

    function remove_kyou_from_multi_column_lists(lists: Array<Array<Kyou>>, deleted_id: string): void {
        for (let i = 0; i < lists.length; i++) {
            remove_kyou_from_list_by_id(lists[i], deleted_id)
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
    function find_reload_query_for(kyou_id: string, data_type: string): FindKyouQuery | undefined {
        // ダイアログを開くたび・フォーカスKyouを引き直すたびに列を総当たりするので、
        // 生の配列を読む(リアクティブProxy越しだと1要素ごとにProxyを作ることになる)。
        for (let i = 0; i < match_kyous_list.value.length; i++) {
            const raw_list = toRaw(match_kyous_list.value[i])
            for (let j = 0; j < raw_list.length; j++) {
                if (raw_list[j].id === kyou_id) {
                    return build_mi_reload_query(querys.value[i], data_type)
                }
            }
        }
        return undefined
    }

    /**
     * @param requested_at 引き直しの合流キー。ポートの変更通知から呼ぶときは
     *   **発生元が採番した値**を渡す。渡さないとここで採番され、
     *   kyou-reload.ts の合流が成立せず画面の枚数ぶん往復する
     */
    async function reload_kyou(kyou: Kyou, requested_at_arg?: number): Promise<void> {
        // 列・focused・開いているダイアログは同じ更新を受けて独立に引き直す。
        // 同じ値を渡して1往復に合流させる（渡さないと系統ごとに往復が増える）
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
                    query: (kyou_in_list) => build_mi_reload_query(column_query, kyou_in_list.data_type),
                })
            }
        })();
        (async (): Promise<void> => {
            if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
                const reload_query = find_reload_query_for(kyou.id, focused_kyou.value.data_type)
                const refreshed = await refresh_kyou(kyou, reload_query, requested_at)
                if (refreshed) {
                    focused_kyou.value = refreshed
                }
            }
        })();
        (async (): Promise<void> => {
            const target_dialogs = opened_dialogs.value.filter(dialog => dialog.kyou.id === kyou.id)
            for (const target_dialog of target_dialogs) {
                const reload_query = find_reload_query_for(kyou.id, target_dialog.kyou.data_type)
                const refreshed = await refresh_kyou(kyou, reload_query, requested_at)
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
            if (is_show_kyou_count_calendar.value || is_show_dnote.value) {
                update_focused_kyous_list(column_index)
            }
            // Dnoteは命令的にreloadするので、配列を触っただけでは追随しない
            reload_dnote_for_column(column_index)
        },
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
            const wait_promises = new Array<Promise<unknown>>()
            try {
                // スクロール位置の復元
                match_kyous_list_top_list.value = props.gkill_api.get_saved_rykv_scroll_indexs(props.column_state_instance_key)

                // 前回開いていた列があれば復元する
                const saved_querys = props.gkill_api.get_saved_rykv_find_kyou_querys(props.column_state_instance_key)
                default_query.value = sidebar.get_default_query()!.clone()
                default_query.value.query_id = props.gkill_api.generate_uuid()
                if (saved_querys.length.valueOf() === 0) {
                    const cloned_default_query = default_query.value.clone()
                    cloned_default_query.query_id = props.gkill_api.generate_uuid()
                    saved_querys.push(cloned_default_query)
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
                if (querys.value[0].calendar_start_date && querys.value[0].calendar_end_date) {
                    gps_log_map_start_time.value = querys.value[0].calendar_start_date!
                    gps_log_map_end_time.value = querys.value[0].calendar_end_date!
                }

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
                nextTick(() => {
                    const refreshed_default_query = query_editor_sidebar.value?.get_default_query()
                    if (refreshed_default_query) {
                        default_query.value = refreshed_default_query.clone()
                    }
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

            // フォーカス列の検索のときだけDnoteを止める。他列の検索で集計中の内容を消さない。
            // 止めるのは実際に検索することが確定した後。deep_equalsの早期returnより前に
            // 止めると、無変更のupdated_query(クリア操作等)でDnoteを消したまま誰も再集計しない
            if (column_index === focused_column_index.value) {
                dnote_reload_seq.value++
                await dnote_view.value?.abort()
            }

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

            props.gkill_api.set_saved_rykv_find_kyou_querys(querys.value, props.column_state_instance_key)

            focused_column_checked_kyous.value = []

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

            // SWキャッシュの掃除は検索1回につき1度だけ
            if (update_cache) {
                await delete_gkill_kyou_cache(null)
            } else {
                await props.gkill_api.delete_updated_gkill_caches()
            }
            if (!is_current()) {
                return
            }

            const base_query = query.clone()
            base_query.parse_words_and_not_words()
            if (update_cache) {
                base_query.update_cache = true
            }

            const req = new GetKyousRequest()
            abort_controllers.set(query_id, req.abort_controller)
            req.query = base_query

            const res: GetKyousResponse = await props.gkill_api.get_kyous(req)

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
                const current_query = querys.value[i]
                if (current_query.query_id === query_id) {
                    column_index = i
                    break
                }
            }

            if (column_index === -1) {
                return
            }

            match_kyous_list.value[column_index] = res.kyous

            if (!props.is_shared_rykv_view) {
                // フォーカス列以外の検索完了でfocused_kyous_listを汚染しない
                if (column_index !== -1 && column_index === focused_column_index.value
                    && (is_show_kyou_count_calendar.value || is_show_dnote.value)) {
                    update_focused_kyous_list(column_index)
                }
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
                if (column_index === focused_column_index.value) {
                    reload_dnote_for_column(column_index)
                }
            })
        } catch (err: unknown) {
            // abortは握りつぶす
            if (!(err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request")))) {
                // abort以外はエラー出力する
                console.error(err)
            }
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
            // focused_queryを差し替える下のfocus_columnだけ
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

            props.gkill_api.set_saved_rykv_find_kyou_querys(querys.value, props.column_state_instance_key)
            props.gkill_api.set_saved_rykv_scroll_indexs(match_kyous_list_top_list.value, props.column_state_instance_key)
            nextTick(() => {
                // 閉じた列の最近傍の列へフォーカスを移す。
                // 削除で-1にしてあるのでfocus_columnが「切り替わった」と判定しDnoteを再集計する
                const next_focused_index = Math.max(0, Math.min(
                    focused_index_before_close > column_index ? focused_index_before_close - 1 : focused_index_before_close,
                    querys.value.length - 1,
                ))
                run_with_sidebar_search_suppressed(() => {
                    focus_column(next_focused_index)
                })
            })
        })
    }

    function add_list_view(query?: FindKyouQuery): void {
        match_kyous_list.value.push(new Array<Kyou>())
        match_kyous_list_top_list.value.push(0)
        // 初期化されていないときはDefaultQueryがない。
        // その場合は初期値のFindKyouQueryをわたして初期化してもらう
        const dq = query_editor_sidebar.value?.get_default_query()?.clone()
        if (query) {
            querys.value.push(query)
            focused_query.value = query
        } else if (dq) {
            dq.query_id = props.gkill_api.generate_uuid()
            querys.value.push(dq)
            focused_query.value = dq
        } else {
            const query = new FindKyouQuery()
            query.query_id = props.gkill_api.generate_uuid()
            querys.value.push(query)
            focused_query.value = query
        }
        if (inited.value) {
            // 列追加もフォーカス切り替えなのでDnoteを追従させる
            focus_column(querys.value.length - 1)
        }
        props.gkill_api.set_saved_rykv_find_kyou_querys(querys.value, props.column_state_instance_key)
        props.gkill_api.set_saved_rykv_scroll_indexs(match_kyous_list_top_list.value, props.column_state_instance_key)
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
            if (inited.value && column_index !== target_column_index) {
                get_kyou_list_view(querys.value[target_column_index].query_id)?.scroll_to_time(kyou.related_time)
            }
        }
    }

    function onFocusedKyouFromSubView(kyou: Kyou): void {
        focused_kyou.value = kyou
        gps_log_map_start_time.value = kyou.related_time
        gps_log_map_end_time.value = kyou.related_time
        gps_log_map_marker_time.value = kyou.related_time
        if (inited.value && kyou_list_views.value) {
            for (let i = 0; i < querys.value.length; i++) {
                if (querys.value[i].is_focus_kyou_in_list_view) {
                    get_kyou_list_view(querys.value[i].query_id)?.scroll_to_time(kyou.related_time)
                }
            }
        }
    }

    async function update_check_kyous(_kyou: Array<Kyou>, _is_checked: boolean): Promise<void> {
        throw new Error('Not implemented')
    }

    // ── Template event handlers ──

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

    async function toggle_dnote(): Promise<void> {
        await dnote_view.value?.abort()
        is_show_dnote.value = !is_show_dnote.value
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
        if (current_query.calendar_start_date && current_query.calendar_end_date) {
            gps_log_map_start_time.value = current_query.calendar_start_date
            gps_log_map_end_time.value = current_query.calendar_end_date
        }
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
        if (new_query.calendar_start_date && new_query.calendar_end_date) {
            gps_log_map_start_time.value = new_query.calendar_start_date
            gps_log_map_end_time.value = new_query.calendar_end_date
        }
    }

    function onColumnScrollList(index: number, scroll_top: number): void {
        // 検索中の列のスクロール通知は、リストを空にした副作用で届く機械的なもの。
        // 取り込むと preserve_scroll の復元先が0で潰れ、そのまま保存位置にも焼き付く
        const column_query = querys.value[index]
        if (column_query && get_kyou_list_view(column_query.query_id)?.get_is_loading()) {
            return
        }
        match_kyous_list_top_list.value[index] = scroll_top
        props.gkill_api.set_saved_rykv_scroll_indexs(match_kyous_list_top_list.value, props.column_state_instance_key)
    }

    function onColumnClickedListView(index: number): void {
        if (props.is_shared_rykv_view) {
            return
        }
        run_with_sidebar_search_suppressed(() => {
            focus_column(index)
        })
    }

    function onColumnClickedKyou(index: number, kyou: Kyou): void {
        run_with_sidebar_search_suppressed(() => {
            focus_column(index)
        })
        clicked_kyou_in_list_view(index, kyou)
        gps_log_map_start_time.value = kyou.related_time
        gps_log_map_end_time.value = kyou.related_time
        gps_log_map_marker_time.value = kyou.related_time
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
            const query = querys.value[index].clone()
            query.is_image_only = is_image_only
            querys.value.splice(index, 1, query)
            querys_backup.value.splice(index, 1, query.clone())
        })
        search(index, querys.value[index], true)
    }

    function onColumnRequestedReloadList(index: number): void {
        const query = querys.value[index].clone()
        querys.value[index] = query
        reload_list(index)
    }

    function onRequestedFocusTime(time: Date): void {
        focused_time.value = time
        gps_log_map_start_time.value = time
        gps_log_map_end_time.value = time
        gps_log_map_marker_time.value = time
    }

    function onGpsLogMapRequestedFocusTime(time: Date): void {
        focused_time.value = time
    }

    function onAddColumnClick(): void {
        // add_list_viewはfocused_queryを差し替えるので抑止で包む
        run_with_sidebar_search_suppressed(() => {
            add_list_view()
        })
        if (props.application_config.rykv_hot_reload) {
            search(querys.value.length - 1, querys.value[querys.value.length - 1], true)
        }
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
            const reload_query = find_reload_query_for(kyou.id, kyou.data_type)
            const refreshed = await refresh_kyou(kyou, reload_query)
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
        'registered_tag': (tag: Tag) => emits('registered_tag', tag),
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

    const subViewFocusHandlers = {
        'focused_kyou': (kyou: Kyou) => onFocusedKyouFromSubView(kyou),
        'clicked_kyou': (kyou: Kyou) => onFocusedKyouFromSubView(kyou),
    }

    const rykv_dialog_handler = {
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
    })

    // ── Keyboard shortcut ──
    // useScopedEnterForKFTL / useScopedCtrlVForClipboard は window にキャプチャで張るので、
    // ポート(rudbeckia)で4画面ぶん登録すると1回の Enter でメモ帳が4枚開く。
    // ホストされているときはポート自身のぶんだけ生かす
    const enable_enter_shortcut = computed(() => !props.is_hosted_in_dialog)
    useScopedEnterForKFTL(rykv_root, show_kftl_dialog, enable_enter_shortcut)
    useScopedCtrlVForClipboard(rykv_root, show_save_clipboard_to_file_dialog, enable_enter_shortcut)

    // ── Return ──
    return {
        // Template refs
        rykv_root,
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
        dnote_view,
        kyou_list_views,
        kyou_detail_view_element,

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
        focused_column_checked_kyous,
        gps_log_map_start_time,
        gps_log_map_end_time,
        gps_log_map_marker_time,
        is_show_kyou_detail_view,
        is_show_kyou_count_calendar,
        is_show_gps_log_map,
        is_show_dnote,
        drawer,
        drawer_mode_is_mobile,
        default_query,
        is_loading,
        inited,
        is_restoring_columns,
        kyou_detail_view_width,

        // Computed
        kyou_list_view_height,
        page_list,
        is_view_ready,

        // Template event handlers
        toggle_drawer,
        navigate_to_page,
        toggle_dnote,
        onSidebarRequestedSearch,
        onSidebarUpdatedQuery,
        onColumnScrollList,
        onColumnClickedListView,
        onColumnClickedKyou,
        onColumnRequestedChangeFocusKyou,
        onColumnRequestedSearch,
        onColumnRequestedChangeImageOnlyView,
        onColumnRequestedReloadList,
        onRequestedFocusTime,
        onGpsLogMapRequestedFocusTime,
        onAddColumnClick,
        onFocusedKyouFromSubView,
        close_list_view,
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
        rykv_dialog_handler,
    }
}
