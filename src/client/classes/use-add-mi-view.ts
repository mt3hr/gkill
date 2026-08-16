import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import { Mi } from '@/classes/datas/mi'
import moment from 'moment'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import { GkillError } from '@/classes/api/gkill-error'
import { AddMiRequest } from '@/classes/api/req_res/add-mi-request'
import { Kyou } from '@/classes/datas/kyou'
import { Notification } from '@/classes/datas/notification'
import { AddNotificationRequest } from '@/classes/api/req_res/add-notification-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { AddMiViewProps } from '@/pages/views/add-mi-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type NewBoardNameDialog from '@/pages/dialogs/new-board-name-dialog.vue'
import type { ComponentRef } from '@/classes/component-ref'
import { useConfirmUnknownMiBoard } from '@/classes/use-confirm-unknown-mi-board'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { add_tags_to_target } from '@/classes/kyou-tags'
import { sort_mi_board_names_by_config_order } from '@/classes/mi-board-names'

export function useAddMiView(options: {
    props: AddMiViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const new_board_name_dialog = ref<InstanceType<typeof NewBoardNameDialog> | null>(null)
    const add_notification_views = ref<ComponentRef | null>(null)
    const kyou_tags_view = ref<ComponentRef | null>(null)

    // ── Confirm unknown mi board ──
    const confirm_unknown_mi_board = useConfirmUnknownMiBoard({ application_config: () => props.application_config })

    // ── Confirm unknown tag ──
    const confirm_unknown_tag = useConfirmUnknownTag({ application_config: () => props.application_config })

    // ── State refs ──
    const is_requested_submit = ref(false)

    const id: Ref<string> = ref(props.gkill_api.generate_uuid())
    const kyou: Ref<Kyou> = computed(() => {
        const k = new Kyou()
        k.id = id.value
        return k
    })
    const mi: Ref<Mi> = ref((() => {
        const m = new Mi()
        m.id = id.value
        return m
    })())
    // APIが返した生の板名一覧。表示順は mi_board_names（設定順に並べ替えたcomputed）を使う
    const mi_board_names_source: Ref<Array<string>> = ref(props.application_config.mi_default_board !== "" ? [props.application_config.mi_default_board] : [])
    // 板の並び順は ApplicationConfig（設定の板構成の「上へ / 下へ」）が正。
    // get_mi_board_list はマップ反復順で返すので、素で渡すと呼ぶたびに並びが変わる。
    // watchではなくcomputedにしているのは、設定ダイアログでの並べ替えのような
    // deepな変更では watch(() => props.application_config, ...) が発火しないため
    const mi_board_names = computed(() => sort_mi_board_names_by_config_order(mi_board_names_source.value, props.application_config.mi_board_struct))

    const mi_title: Ref<string> = ref(mi.value ? mi.value.title : "")
    const mi_board_name: Ref<string> = ref(props.application_config.mi_default_board !== "" ? props.application_config.mi_default_board : "Inbox")
    const mi_estimate_start_date_typed: Ref<Date | null> = ref(mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).toDate() : null)
    const mi_estimate_start_date_string: Ref<string> = computed(() => mi_estimate_start_date_typed.value ? moment(mi_estimate_start_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_estimate_start_time_string: Ref<string> = ref(mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("HH:mm:ss") : "")
    const mi_estimate_end_date_typed: Ref<Date | null> = ref(mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).toDate() : null)
    const mi_estimate_end_date_string: Ref<string> = computed(() => mi_estimate_end_date_typed.value ? moment(mi_estimate_end_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_estimate_end_time_string: Ref<string> = ref(mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("HH:mm:ss") : "")
    const mi_limit_date_typed: Ref<Date | null> = ref(mi.value && mi.value.limit_time ? moment(mi.value.limit_time).toDate() : null)
    const mi_limit_date_string: Ref<string> = computed(() => mi_limit_date_typed.value ? moment(mi_limit_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_limit_time_string: Ref<string> = ref(mi.value && mi.value.limit_time ? moment(mi.value.limit_time).format("HH:mm:ss") : "")

    const notifications: Ref<Array<Notification>> = ref(new Array<Notification>())
    const show_start_date_menu = ref(false)
    const show_start_time_menu = ref(false)
    const show_end_date_menu = ref(false)
    const show_end_time_menu = ref(false)
    const show_limit_date_menu = ref(false)
    const show_limit_time_menu = ref(false)

    // ── Watchers ──
    watch(() => props.application_config, () => load_mi_board_names())

    // ── Init ──
    load_mi_board_names()

    // ── Business logic ──
    async function load_mi_board_names(): Promise<void> {
        const req = new GetMiBoardRequest()

        const res = await props.gkill_api.get_mi_board_list(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            // emits('received_messages', res.messages)
        }

        let is_contain_default_board = false
        res.boards.forEach((board_name) => {
            if (board_name === props.application_config.mi_default_board) {
                is_contain_default_board = true
            }
        })
        if (!is_contain_default_board) {
            res.boards.push(props.application_config.mi_default_board)
        }

        mi_board_names_source.value = res.boards
    }

    function update_board_name(board_name: string): void {
        mi_board_names_source.value.push(board_name)
        mi_board_name.value = board_name
    }

    function show_new_board_name_dialog(): void {
        new_board_name_dialog.value?.show()
    }

    function clear_estimate_start_date_time(): void {
        mi_estimate_start_date_typed.value = null
        mi_estimate_start_time_string.value = ""
    }

    function clear_estimate_end_date_time(): void {
        mi_estimate_end_date_typed.value = null
        mi_estimate_end_time_string.value = ""
    }

    function clear_limit_date_time(): void {
        mi_limit_date_typed.value = null
        mi_limit_time_string.value = ""
    }

    function now_to_estimate_start_date_time(): void {
        mi_estimate_start_date_typed.value = moment().toDate()
        mi_estimate_start_time_string.value = moment().format("HH:mm:ss")
    }

    function now_to_estimate_end_date_time(): void {
        mi_estimate_end_date_typed.value = moment().toDate()
        mi_estimate_end_time_string.value = moment().format("HH:mm:ss")
    }

    function now_to_limit_date_time(): void {
        mi_limit_date_typed.value = moment().toDate()
        mi_limit_time_string.value = moment().format("HH:mm:ss")
    }

    function reset(): void {
        mi_title.value = mi.value.title
        mi_board_name.value = props.application_config.mi_default_board
        mi_estimate_start_date_typed.value = mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).toDate() : null
        mi_estimate_start_time_string.value = mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("HH:mm:ss") : ""
        mi_estimate_end_date_typed.value = mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).toDate() : null
        mi_estimate_end_time_string.value = mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("HH:mm:ss") : ""
        mi_limit_date_typed.value = mi.value && mi.value.limit_time ? moment(mi.value.limit_time).toDate() : null
        mi_limit_time_string.value = mi.value && mi.value.limit_time ? moment(mi.value.limit_time).format("HH:mm:ss") : ""
        notifications.value.splice(0)
        kyou_tags_view.value?.reset()
    }

    async function save(): Promise<void> {
        await do_save(false, false)
    }

    /**
     * 保存の本体。確認を挟むたびにここへ戻ってくる。
     *
     * 確認は「タグ → 板名」の順に1つずつ出す（KFTLの do_submit と同じ）。
     * 確認ダイアログは非モーダルなので、確認中に入力を変えられる。
     * 再入のたびに Notification もタグ名も取り直すこと
     */
    async function do_save(skip_unknown_tag_check: boolean, skip_unknown_mi_board_check: boolean): Promise<void> {
        try {
            is_requested_submit.value = true
            // Notification チェック
            // おかしかったらnullが戻ってくるので中断する
            const notification_results = new Array<Notification>()
            if (add_notification_views.value) {
                for (let i = 0; i < add_notification_views.value.length; i++) {
                    const notification = await add_notification_views.value[i].get_notification()
                    if (!notification) {
                        return
                    }
                    notification_results.push(notification)
                }
            }

            // Mi チェック
            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            if (!mi.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_mi_is_null
                error.error_message = i18n.global.t("CLIENT_MI_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // タイトルの入力チェック
            if (mi_title.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.mi_title_is_blank
                error.error_message = i18n.global.t("MI_TITLE_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 開始日時 片方だけ入力されていたらエラーチェック
            if (mi_estimate_start_date_string.value === "" || mi_estimate_start_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
                if ((mi_estimate_start_date_string.value === "" && mi_estimate_start_time_string.value !== "") ||
                    (mi_estimate_start_date_string.value !== "" && mi_estimate_start_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                    const error = new GkillError()
                    error.error_code = GkillErrorCodes.mi_estimate_start_time_is_blank
                    error.error_message = i18n.global.t("MI_START_DATE_TIME_IS_BLANK_MESSAGE")
                    const errors = new Array<GkillError>()
                    errors.push(error)
                    emits('received_errors', errors)
                    return
                }
            }

            // 終了日時 片方だけ入力されていたらエラーチェック
            if (mi_estimate_end_date_string.value === "" || mi_estimate_end_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
                if ((mi_estimate_end_date_string.value === "" && mi_estimate_end_time_string.value !== "") ||
                    (mi_estimate_end_date_string.value !== "" && mi_estimate_end_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                    const error = new GkillError()
                    error.error_code = GkillErrorCodes.mi_estimate_end_time_is_blank
                    error.error_message = i18n.global.t("MI_END_DATE_TIME_IS_BLANK_MESSAGE")
                    const errors = new Array<GkillError>()
                    errors.push(error)
                    emits('received_errors', errors)
                    return
                }
            }

            // 期限日時 片方だけ入力されていたらエラーチェック
            if (mi_limit_date_string.value === "" || mi_limit_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
                if ((mi_limit_date_string.value === "" && mi_limit_time_string.value !== "") ||
                    (mi_limit_date_string.value !== "" && mi_limit_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                    const error = new GkillError()
                    error.error_code = GkillErrorCodes.mi_limit_time_is_blank
                    error.error_message = i18n.global.t("MI_LIMIT_DATE_TIME_IS_BLANK_MESSAGE")
                    const errors = new Array<GkillError>()
                    errors.push(error)
                    emits('received_errors', errors)
                    return
                }
            }

            // 更新がなかったらエラーメッセージを出力する
            if (mi.value.title === mi_title.value &&
                moment(mi.value.estimate_start_time).valueOf() === moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value).valueOf() &&
                moment(mi.value.estimate_end_time).valueOf() === moment(mi_estimate_end_date_string.value + " " + mi_estimate_end_time_string.value).valueOf() &&
                moment(mi.value.limit_time).valueOf() === moment(mi_limit_date_string.value + " " + mi_limit_time_string.value).valueOf()
            ) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.mi_is_no_update
                error.error_message = i18n.global.t("MI_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // タグツリーに無いタグ名なら、保存する前に確認を取る
            const tag_names = kyou_tags_view.value?.get_tag_names() ?? []
            if (!skip_unknown_tag_check) {
                const unknown_tags = confirm_unknown_tag.collect_unknown_tags(tag_names)
                if (unknown_tags.length !== 0) {
                    confirm_unknown_tag.open_confirm(unknown_tags)
                    return
                }
            }

            // 実在しない板名なら、保存する前に確認を取る。
            // 板はサーバ側で検証されず「その名前のタスクが1件でもあること」で実体化するので、
            // 打ち間違いがそのまま新しい板になってしまう。
            // タグの確認を通した後に改めてここへ来る（確認は1つずつ順に出す）
            if (!skip_unknown_mi_board_check) {
                const unknown_boards = confirm_unknown_mi_board.collect_unknown_mi_boards([mi_board_name.value])
                if (unknown_boards.length !== 0) {
                    confirm_unknown_mi_board.open_confirm(unknown_boards)
                    return
                }
            }

            await execute_save(notification_results, tag_names)
        } finally {
            is_requested_submit.value = false
        }
    }

    function cancel_tag_save(): void {
        confirm_unknown_tag.close_confirm()
    }

    /** タグの確認を通した。次は板名の確認へ進む */
    async function confirm_tag_save(): Promise<void> {
        confirm_unknown_tag.close_confirm()
        await do_save(true, false)
    }

    function cancel_save(): void {
        confirm_unknown_mi_board.close_confirm()
    }

    async function confirm_save(): Promise<void> {
        confirm_unknown_mi_board.remember_confirmed_mi_boards()
        confirm_unknown_mi_board.close_confirm()
        await do_save(true, true)
    }

    async function execute_save(notification_results: Array<Notification>, tag_names: Array<string>): Promise<void> {
        try {
            is_requested_submit.value = true
            if (!mi.value) {
                return
            }

            // Mi 追加

            // 作成するMi情報を用意する
            let estimate_start_time: Date | null = null
            let estimate_end_time: Date | null = null
            let limit_time: Date | null = null
            if (mi_estimate_start_date_string.value !== "" && mi_estimate_start_time_string.value !== "") {
                estimate_start_time = moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value).toDate()
            }
            if (mi_estimate_end_date_string.value !== "" && mi_estimate_end_time_string.value !== "") {
                estimate_end_time = moment(mi_estimate_end_date_string.value + " " + mi_estimate_end_time_string.value).toDate()
            }
            if (mi_limit_date_string.value !== "" && mi_limit_time_string.value !== "") {
                limit_time = moment(mi_limit_date_string.value + " " + mi_limit_time_string.value).toDate()
            }
            const new_mi = mi.value.clone()
            new_mi.id = mi.value.id
            new_mi.title = mi_title.value
            new_mi.board_name = mi_board_name.value
            new_mi.estimate_start_time = estimate_start_time
            new_mi.estimate_end_time = estimate_end_time
            new_mi.limit_time = limit_time
            new_mi.create_app = "gkill"
            new_mi.create_device = props.application_config.device
            new_mi.create_time = new Date(Date.now())
            new_mi.create_user = props.application_config.user_id
            new_mi.update_app = "gkill"
            new_mi.update_device = props.application_config.device
            new_mi.update_time = new Date(Date.now())
            new_mi.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_mi.id)
            const req = new AddMiRequest()
            req.mi = new_mi
            req.want_response_kyou = true
            const res = await props.gkill_api.add_mi(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }

            // Notification 追加
            for (let i = 0; i < notification_results.length; i++) {
                // 追加リクエストを飛ばす
                await delete_gkill_kyou_cache(notification_results[i].id)
                const notif_req = new AddNotificationRequest()
                notif_req.notification = notification_results[i]
                const notif_res = await props.gkill_api.add_notification(notif_req)
                if (notif_res.errors && notif_res.errors.length !== 0) {
                    emits('received_errors', notif_res.errors)
                    return
                }
                if (notif_res.messages && notif_res.messages.length !== 0) {
                    emits('received_messages', notif_res.messages)
                }
            }

            // タグは registered_kyou より必ず先に付ける。
            // 先に emit すると、タグで絞り込んだ列が空のタグ列を見て「一致しない」と判定し、
            // エラーも出ないまま行が現れない
            const tag_result = await add_tags_to_target(props.gkill_api, props.application_config, new_mi.id, tag_names)
            tag_result.added_tags.forEach(added_tag => emits('registered_tag', added_tag))
            if (tag_result.messages.length !== 0) {
                emits('received_messages', tag_result.messages)
            }
            if (tag_result.errors.length !== 0) {
                emits('received_errors', tag_result.errors)
            }

            // 追加した記録は列へ局所挿入されるので、リスト全体の引き直しは要求しない。
            // Kyouが返らなかったときだけ、従来どおり引き直しへ落とす
            if (res.added_kyou) {
                emits("registered_kyou", res.added_kyou)
            } else {
                emits('requested_reload_list')
            }
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    function add_notification(): void {
        const notification = new Notification()
        notification.id = props.gkill_api.generate_uuid()
        notification.target_id = id.value
        notification.content = mi_title.value
        notification.notification_time = new Date(0)
        if (mi_estimate_start_date_string.value !== "" && mi_estimate_start_time_string.value !== "") {
            notification.notification_time = moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value).toDate()
        }
        notifications.value.push(notification)
    }

    function delete_notification(index: number): void {
        notifications.value.splice(index, 1)
    }

    // ── CRUD relay handlers ──
    const crudRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    // ── Second init call ──
    load_mi_board_names()

    // ── Return ──
    return {
        // Template refs
        new_board_name_dialog,
        add_notification_views,
        kyou_tags_view,
        confirm_unknown_mi_board_dialog: confirm_unknown_mi_board.confirm_unknown_mi_board_dialog,
        confirm_unknown_tag_dialog: confirm_unknown_tag.confirm_unknown_tag_dialog,

        // Confirm unknown mi board
        unknown_mi_boards: confirm_unknown_mi_board.unknown_mi_boards,
        cancel_save,
        confirm_save,

        // Confirm unknown tag
        unknown_tags: confirm_unknown_tag.unknown_tags,
        cancel_tag_save,
        confirm_tag_save,

        // State
        is_requested_submit,
        id,
        kyou,
        mi,
        mi_board_names,
        mi_title,
        mi_board_name,
        mi_estimate_start_date_typed,
        mi_estimate_start_date_string,
        mi_estimate_start_time_string,
        mi_estimate_end_date_typed,
        mi_estimate_end_date_string,
        mi_estimate_end_time_string,
        mi_limit_date_typed,
        mi_limit_date_string,
        mi_limit_time_string,
        notifications,
        show_start_date_menu,
        show_start_time_menu,
        show_end_date_menu,
        show_end_time_menu,
        show_limit_date_menu,
        show_limit_time_menu,

        // Business logic / template handlers
        load_mi_board_names,
        update_board_name,
        show_new_board_name_dialog,
        clear_estimate_start_date_time,
        clear_estimate_end_date_time,
        clear_limit_date_time,
        now_to_estimate_start_date_time,
        now_to_estimate_end_date_time,
        now_to_limit_date_time,
        reset,
        save,
        add_notification,
        delete_notification,

        // Event relay objects
        crudRelayHandlers,
    }
}
