import { i18n } from '@/i18n'
import { gkill_page_list } from '@/classes/gkill-page-list'
import router from '@/router'
import { computed, onUnmounted, ref, type Ref } from 'vue'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { DashboardConfig } from '@/classes/datas/config/dashboard-config'
import { Kyou } from '@/classes/datas/kyou'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import type { ComponentRef } from '@/classes/component-ref'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import moment from 'moment'
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'
import { useScopedCtrlVForClipboard } from '@/classes/use-scoped-ctrl-v-for-clipboard'
import { build_kyou_dialog_host_handlers } from '@/classes/kyou-view-relay'
import { new_reload_batch, refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import type { KyouChange } from '@/classes/kyou-change-bus'
import { useKyouChangeSubscriber } from '@/classes/use-kyou-change-subscriber'
import { can_decide_query_locally, decide_local_insert, insert_kyou_sorted } from '@/classes/kyou-local-insert'
import { hydrate } from '@/classes/api/hydrate'
import type { Tag } from '@/classes/datas/tag'
// 中継束が扱うのは gkill のデータクラス。import し忘れると DOM のグローバルを指す
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { DashboardViewProps } from '@/pages/views/dashboard-view-props'
import type { DashboardViewEmits } from '@/pages/views/dashboard-view-emits'

/**
 * ダッシュボードの本体。
 *
 * ページ(use-dashboard-page.ts)からは ApplicationConfig の取得・テーマ・
 * ウィンドウのリサイズ購読・メッセージ表示・板/タグツリーの追随・ログアウトだけが残り、
 * 中身はすべてこちらに居る。rykv / mi と同じ「ページは薄いラッパ、ビューが本体」の形。
 * こうしておくとポート(rudbeckia)のフローティングダイアログの中にも同じものを置ける。
 */
export function useDashboardView(options: {
    props: DashboardViewProps,
    emits: DashboardViewEmits,
    /**
     * requested_reload_list / registered_kyou を受けたときに全体を取り直す処理。
     * DnoteView の reload() はテンプレートref経由でしか呼べないので、実体は .vue 側が渡す
     */
    reload_all?: () => Promise<void>,
    /**
     * Dnote だけを取り直す処理。
     * Kyou 1件の追加は Mi リストへ局所挿入できるが、Dnote は集計なので取り直すしかない
     */
    reload_dnote?: () => Promise<void>,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const dashboard_root = ref<HTMLElement | null>(null)
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

    // ── State refs ──
    const selected_date: Ref<Date> = ref(moment().startOf('day').toDate())
    const checked_kyous: Ref<Array<Kyou>> = ref([])
    const mi_kyous: Ref<Array<Kyou>> = ref([])
    const dnote_kyous: Ref<Array<Kyou>> = ref([])
    let mi_kyous_fetch_epoch = 0
    let current_mi_abort_controller: AbortController | null = null
    let current_dnote_abort_controller: AbortController | null = null
    const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])

    const gkill_api = computed(() => props.gkill_api)
    const application_config = computed(() => props.application_config)

    // ── Computed ──
    // 画面切替メニューの一覧は classes/gkill-page-list.ts に1つだけ置いてある
    const page_list = gkill_page_list

    const panel_height = computed<number>(() => Math.max(400, props.app_content_height.valueOf() - 350))

    const target_date_start = computed<Date>(() => moment(selected_date.value).startOf('day').toDate())
    const target_date_end = computed<Date>(() => moment(selected_date.value).endOf('day').toDate())

    const date_label = computed<string>(() => {
        const weekday_keys = [
            'SUNDAY_TITLE', 'MONDAY_TITLE', 'TUESDAY_TITLE', 'WEDNESDAY_TITLE',
            'THURSDAY_TITLE', 'FRIDAY_TITLE', 'SATURDAY_TITLE',
        ]
        const day_of_week = i18n.global.t(weekday_keys[moment(selected_date.value).day()])
        return `${moment(selected_date.value).format('YYYY/M/D')}(${day_of_week})`
    })

    const date_picker_model = computed<Date>({
        get: () => selected_date.value,
        set: (value: Date) => {
            selected_date.value = moment(value).startOf('day').toDate()
        },
    })

    const dnote_query = computed<FindKyouQuery>(() => {
        const base_query = new FindKyouQuery()
        // rep/tagフィルタは既定で未使用(null)。保存済み条件があればそちらを採用する
        base_query.reps = null
        base_query.tags = null
        if (application_config.value.dashboard_json_data) {
            const config = DashboardConfig.parse(application_config.value.dashboard_json_data)
            if (config.dashboard_dnote_find_kyou_query) {
                const saved = config.dashboard_dnote_find_kyou_query
                base_query.tags = saved.tags === null ? null : saved.tags.concat()
                base_query.tags_and = saved.tags_and
                base_query.reps = saved.reps === null ? null : saved.reps.concat()
                base_query.keywords = saved.keywords
                base_query.words = saved.words === null ? null : saved.words.concat()
                base_query.not_words = saved.not_words === null ? null : saved.not_words.concat()
            }
        }
        base_query.calendar_start_date = target_date_start.value
        base_query.calendar_end_date = target_date_end.value
        base_query.apply_hide_tags(application_config.value)
        return base_query
    })

    const mi_kyou_query = computed<FindKyouQuery>(() => {
        const query = new FindKyouQuery()
        if (application_config.value.dashboard_json_data) {
            const config = DashboardConfig.parse(application_config.value.dashboard_json_data)
            if (config.dashboard_mi_find_kyou_query) {
                const saved = config.dashboard_mi_find_kyou_query
                query.include_create_mi = saved.include_create_mi
                query.include_check_mi = saved.include_check_mi
                query.include_limit_mi = saved.include_limit_mi
                query.include_start_mi = saved.include_start_mi
                query.include_end_mi = saved.include_end_mi
                query.mi_sort_type = saved.mi_sort_type
                query.tags = saved.tags === null ? null : saved.tags.concat()
                query.tags_and = saved.tags_and
                query.keywords = saved.keywords
                query.words = saved.words === null ? null : saved.words.concat()
                query.not_words = saved.not_words === null ? null : saved.not_words.concat()
                query.mi_check_state = saved.mi_check_state
            }
        }
        query.for_mi = true
        query.include_limit_mi = true
        query.reps = null
        query.calendar_start_date = target_date_start.value
        query.calendar_end_date = target_date_end.value
        query.apply_hide_tags(application_config.value)
        return query
    })

    // ── Loading state ──
    const is_loading = ref(true)
    // パネルのデータ取得が飛行中。表示制御には使わず、E2Eの準備完了信号にだけ使う
    const is_fetching = ref(false)

    // 「操作してよい状態」をE2Eが決定論的に待つための信号。
    // 画面の表示/非表示には一切使わない(使うと取得完了を待たなくした意味がなくなる)
    const is_view_ready = computed(() => application_config.value.is_loaded && !is_fetching.value)

    // ── Business logic ──
    function abort_all_fetches(): void {
        if (current_mi_abort_controller) {
            current_mi_abort_controller.abort()
            current_mi_abort_controller = null
        }
        if (current_dnote_abort_controller) {
            current_dnote_abort_controller.abort()
            current_dnote_abort_controller = null
        }
        mi_kyous_fetch_epoch++
    }

    function clear_dashboard_datas(): void {
        checked_kyous.value.splice(0)
        dnote_kyous.value.splice(0)
        mi_kyous.value.splice(0)
    }

    async function fetch_mi_kyous(): Promise<void> {
        const current_epoch = ++mi_kyous_fetch_epoch
        const req = new GetKyousRequest()
        req.query = mi_kyou_query.value
        current_mi_abort_controller = req.abort_controller

        try {
            const res = await gkill_api.value.get_kyous(req)
            if (current_epoch !== mi_kyous_fetch_epoch) {
                return
            }
            if (res.errors && res.errors.length !== 0) {
                write_errors(res.errors)
                return
            }
            mi_kyous.value = res.kyous
        } catch (_e: unknown) {
            // abort時は無視
        }
    }

    async function fetch_dnote_kyous(): Promise<Array<Kyou>> {
        if (current_dnote_abort_controller) {
            current_dnote_abort_controller.abort()
        }
        const req = new GetKyousRequest()
        req.query = dnote_query.value
        current_dnote_abort_controller = req.abort_controller
        try {
            const res = await gkill_api.value.get_kyous(req)
            if (res.errors && res.errors.length !== 0) {
                write_errors(res.errors)
                return []
            }
            dnote_kyous.value = res.kyous
            return res.kyous
        } catch (_e: unknown) {
            // abort時は無視
            return []
        }
    }

    // メッセージの表示はページが持っている。ビューは上げるだけ
    function write_errors(errors: Array<GkillError>): void {
        emits('received_errors', errors)
    }

    function write_messages(messages: Array<GkillMessage>): void {
        emits('received_messages', messages)
    }

    // ── Rykv dialog ──
    function open_rykv_dialog(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload): void {
        const dialog_id = gkill_api.value.generate_uuid()
        opened_dialogs.value.push({
            id: dialog_id,
            kind,
            kyou: kyou.clone(),
            payload: payload ?? null,
            opened_at: Date.now(),
        });
        // 開いた直後にも最新化する。リストのKyouは検索時点のものなので、
        // 別経路で更新されていると古い内容でダイアログが開いてしまう
        (async (): Promise<void> => {
            const refreshed = await refresh_kyou(kyou, mi_kyou_query.value)
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

    // ── Kyou update propagation ──
    // ダッシュボードは Mi リスト / Dnote / 開いているダイアログの3箇所にKyouを抱えている。
    // 以前は RykvDialogHost に closed / received_errors / received_messages しか配線しておらず、
    // Kyouもタグもどう編集しても、どこも（開いているダイアログ自身すら）更新されなかった
    /**
     * @param requested_at_arg 引き直しの合流キー。ポートの変更通知から呼ぶときは
     *   **発生元が採番した値**を渡す
     */
    async function reload_kyou(kyou: Kyou, requested_at_arg?: number): Promise<void> {
        // 再検索ではなく該当Kyouだけ差し替える。再検索するとソート順もヒット集合も変わって
        // スクロール位置が飛ぶし、KyouListView は配列参照の差し替えでフル再描画する。
        // 再検索したいときは requested_reload_list という別のイベントが飛んでくる
        // 3ブロックは同じ更新から派生しているので、同じ値を渡して1往復に合流させる
        const requested_at = requested_at_arg ?? new_reload_batch()
        await refresh_kyou_in_list(mi_kyous.value, kyou, {
            requested_at: requested_at,
            query: mi_kyou_query.value,
            replace: (next_list) => { mi_kyous.value = next_list },
        })
        await refresh_kyou_in_list(checked_kyous.value, kyou, { requested_at: requested_at })
        for (let i = 0; i < opened_dialogs.value.length; i++) {
            if (opened_dialogs.value[i].kyou.id === kyou.id) {
                const refreshed = await refresh_kyou(kyou, mi_kyou_query.value, requested_at)
                if (refreshed) {
                    opened_dialogs.value[i] = { ...opened_dialogs.value[i], kyou: refreshed }
                }
            }
        }
    }

    function remove_kyou_by_id(list: Array<Kyou>, deleted_id: string): void {
        for (let i = list.length - 1; i >= 0; i--) {
            if (list[i].id === deleted_id) {
                list.splice(i, 1)
            }
        }
    }

    function onDeletedKyou(deleted_kyou: Kyou): void {
        remove_kyou_by_id(mi_kyous.value, deleted_kyou.id)
        remove_kyou_by_id(dnote_kyous.value, deleted_kyou.id)
        remove_kyou_by_id(checked_kyous.value, deleted_kyou.id)
    }

    function update_check_kyous(kyous: Array<Kyou>, is_checked: boolean): void {
        for (let i = 0; i < kyous.length; i++) {
            const index = checked_kyous.value.findIndex(checked_kyou => checked_kyou.id === kyous[i].id)
            if (is_checked && index < 0) {
                checked_kyous.value.push(kyous[i])
            } else if (!is_checked && index >= 0) {
                checked_kyous.value.splice(index, 1)
            }
        }
    }

    async function reload_all(): Promise<void> {
        if (options?.reload_all) {
            await options.reload_all()
            return
        }
        await Promise.all([fetch_mi_kyous(), fetch_dnote_kyous()])
    }

    async function reload_dnote(): Promise<void> {
        if (options?.reload_dnote) {
            await options.reload_dnote()
            return
        }
        await fetch_dnote_kyous()
    }

    // KFTLで複数行を一度に投げると registered_kyou が連続発火するのでまとめる。
    // 予約は引き直しのawaitより後に来ることがあるので、アンマウント後は受け付けない
    // (cancelだけでは、離脱後に決着した引き直しが張ったタイマーを取り逃がす)
    let is_unmounted = false
    const reload_debounce_milli_seconds = 300
    function make_debounced(run: () => void): { schedule: () => void, cancel: () => void } {
        let timer: ReturnType<typeof setTimeout> | null = null
        return {
            schedule: () => {
                if (is_unmounted) {
                    return
                }
                if (timer) {
                    clearTimeout(timer)
                }
                timer = setTimeout(() => {
                    timer = null
                    run()
                }, reload_debounce_milli_seconds)
            },
            cancel: () => {
                if (timer) {
                    clearTimeout(timer)
                    timer = null
                }
            },
        }
    }
    const reload_all_debounce = make_debounced(() => { reload_all() })
    const reload_dnote_debounce = make_debounced(() => { reload_dnote() })
    onUnmounted(() => {
        is_unmounted = true
        reload_all_debounce.cancel()
        reload_dnote_debounce.cancel()
        // 飛行中の取得を止める。ポート(rudbeckia)ではこのビューごと閉じられる
        abort_all_fetches()
    })

    /**
     * 追加されたKyouを、再検索せずにMiリストへ差し込む。
     * Dnoteは集計なので差し込めず、まとめて取り直す。
     * Miリストの条件をクライアントで判定しきれないときだけ、従来どおり全体を取り直す。
     */
    async function insert_registered_kyou(raw: unknown, requested_at_arg?: number): Promise<void> {
        reload_dnote_debounce.schedule()

        const query = mi_kyou_query.value
        if (!can_decide_query_locally(query).ok) {
            reload_all_debounce.schedule()
            return
        }
        // add_* の応答は hydrate を通っていない生JSONなので実体化する
        const kyou = raw instanceof Kyou ? raw : hydrate(new Kyou(), raw)
        if (!kyou.id) {
            return
        }
        const refreshed = await refresh_kyou(kyou, query, requested_at_arg ?? new_reload_batch())
        if (!refreshed || !refreshed.id) {
            reload_all_debounce.schedule()
            return
        }
        if (refreshed.is_deleted) {
            return
        }
        const decision = decide_local_insert(refreshed.clone(), query)
        if (decision.kind === 'undecidable') {
            reload_all_debounce.schedule()
            return
        }
        if (decision.kind === 'skip') {
            return
        }
        // ダッシュボードは他所も copy-on-write で反応性を飛ばしているので合わせる
        const next_list = mi_kyous.value.concat()
        let is_mutated = false
        for (const row of decision.rows) {
            is_mutated = insert_kyou_sorted(next_list, row, query) || is_mutated
        }
        if (is_mutated) {
            mi_kyous.value = next_list
        }
    }

    // ── Event relay objects ──
    // RykvDialogHost / DnoteView / KyouListView の3箇所に同じ束を渡す。
    // closed を出さないコンポーネントに渡っても発火しないので無害。
    // 板ツリー/タグツリーの追随はページが持っているので、ローカルの反映に加えて上へも emit する
    const dashboardKyouHandlers = build_kyou_dialog_host_handlers({
        'closed': (dialog_id: string) => close_rykv_dialog(dialog_id),
        'updated_kyou': (kyou: Kyou) => {
            // 引き直しの合流キーはここで採番して、通知にも同じ値を載せる
            const requested_at = new_reload_batch()
            reload_kyou(kyou, requested_at)
            emits('updated_kyou', kyou)
            publish_kyou_change({ kind: 'reload', kyou: kyou }, requested_at)
        },
        'deleted_kyou': (kyou: Kyou) => {
            onDeletedKyou(kyou)
            emits('deleted_kyou', kyou)
            publish_kyou_change({ kind: 'deleted', kyou: kyou }, new_reload_batch())
        },
        // タグ/テキスト/通知の変更はこれしか出さない。配らないと他の画面に一切届かない
        'requested_reload_kyou': (kyou: Kyou) => {
            const requested_at = new_reload_batch()
            reload_kyou(kyou, requested_at)
            publish_kyou_change({ kind: 'reload', kyou: kyou }, requested_at)
        },
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => open_rykv_dialog(kind, kyou, payload),
    }, {
        'received_errors': (errors: Array<GkillError>) => write_errors(errors),
        'received_messages': (received_messages: Array<GkillMessage>) => write_messages(received_messages),
        'requested_reload_list': () => {
            reload_all()
            publish_kyou_change({ kind: 'reload_list' }, new_reload_batch())
        },
        'requested_update_check_kyous': (kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked),
        'registered_kyou': (kyou: Kyou) => {
            const requested_at = new_reload_batch()
            insert_registered_kyou(kyou, requested_at).catch((err: unknown) => console.error(err))
            emits('registered_kyou', kyou)
            publish_kyou_change({ kind: 'registered', kyou: kyou }, requested_at)
        },
        'registered_tag': (tag: Tag) => emits('registered_tag', tag),
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'deleted_tag': (tag: Tag) => emits('deleted_tag', tag),
        'registered_text': (text: Text) => emits('registered_text', text),
        'updated_text': (text: Text) => emits('updated_text', text),
        'deleted_text': (text: Text) => emits('deleted_text', text),
        'registered_notification': (notification: Notification) => emits('registered_notification', notification),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'deleted_notification': (notification: Notification) => emits('deleted_notification', notification),
    })

    // KFTL/MKFL ダイアログはタグを registered_tag で上げてこないので、保存完了で両方取り直す。
    // 取り直しはページの useConfigStructSync が持っているので上げるだけ
    function onSavedKyouByKftl(last_added_request_time: Date): void {
        emits('saved_kyou_by_kftl', last_added_request_time)
    }

    // ── Navigation ──
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

    // ── Floating button ──
    function floating_action_button_style() {
        return {
            'bottom': '60px',
            'right': '10px',
            'height': '50px',
            'width': '50px',
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

    // ── Date navigation ──
    function go_prev_day(): void {
        selected_date.value = moment(selected_date.value).subtract(1, 'day').startOf('day').toDate()
    }
    function go_next_day(): void {
        selected_date.value = moment(selected_date.value).add(1, 'day').startOf('day').toDate()
    }
    function go_today(): void {
        selected_date.value = moment().startOf('day').toDate()
    }

    // ── 画面間の変更通知 ──
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
        apply_deleted: (kyou) => onDeletedKyou(kyou),
        apply_reload_list: () => { void reload_all() },
    })

    // ── Keyboard shortcuts ──
    // window にキャプチャで張るので、ポート(rudbeckia)で複数枚ホストすると多重登録になる
    const enable_enter_shortcut = computed(() => !props.is_hosted_in_dialog)
    useScopedEnterForKFTL(dashboard_root, show_kftl_dialog, enable_enter_shortcut)
    useScopedCtrlVForClipboard(dashboard_root, show_save_clipboard_to_file_dialog, enable_enter_shortcut)

    // ── Return ──
    return {
        // Template refs
        dashboard_root,
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

        // State
        is_loading,
        is_fetching,
        gkill_api,
        application_config,
        selected_date,
        checked_kyous,
        mi_kyous,
        dnote_kyous,
        opened_dialogs,

        // Computed
        is_view_ready,
        panel_height,
        page_list,
        target_date_start,
        target_date_end,
        date_label,
        date_picker_model,
        dnote_query,
        mi_kyou_query,

        // Methods
        write_errors,
        write_messages,
        navigate_to_page,
        abort_all_fetches,
        clear_dashboard_datas,
        fetch_mi_kyous,
        fetch_dnote_kyous,
        go_prev_day,
        go_next_day,
        go_today,
        floating_action_button_style,
        show_kftl_dialog,
        show_add_kc_dialog,
        show_mkfl_dialog,
        show_timeis_dialog,
        show_mi_dialog,
        show_nlog_dialog,
        show_lantana_dialog,
        show_urlog_dialog,
        show_upload_file_dialog,
        show_save_clipboard_to_file_dialog,
        open_rykv_dialog,
        close_rykv_dialog,
        reload_kyou,
        insert_registered_kyou,
        reload_dnote,

        // Event relay objects
        dashboardKyouHandlers,
        onSavedKyouByKftl,
    }
}
