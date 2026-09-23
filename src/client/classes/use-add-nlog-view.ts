import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import { GkillError } from '@/classes/api/gkill-error'
import moment from 'moment'
import { Nlog } from '@/classes/datas/nlog'
import { AddNlogRequest } from '@/classes/api/req_res/add-nlog-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { AddNlogViewProps } from '@/pages/views/add-nlog-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { ComponentRef } from '@/classes/component-ref'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { add_tags_to_target, record_added_tag_history } from '@/classes/kyou-tags'
import { fetch_committed_kyou, run_in_tx } from '@/classes/gkill-tx'
import type { Tag } from '@/classes/datas/tag'
import type { Kyou } from '@/classes/datas/kyou'

/**
 * 支出の入力の1行（品名と金額の組）。メモ帳（KFTL）の支出と同じく、店名と日時は全行で共有し、
 * 1行が1件の支出になる。
 */
export interface AddNlogRowInput {
    // v-for の key。保存する Nlog の id とは別（id は保存するときに行ごとに採る）
    row_key: number
    title: string
    // v-text-field（type="number"）は文字列を返すので、保存の直前に数値へ直す
    amount: number | string
}

export function useAddNlogView(options: {
    props: AddNlogViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_tags_view = ref<ComponentRef | null>(null)

    // ── Confirm unknown tag ──
    const confirm_unknown_tag = useConfirmUnknownTag({ application_config: () => props.application_config })

    // ── State refs ──
    const is_requested_submit = ref(false)

    const nlog: Ref<Nlog> = ref((() => {
        const nlog = new Nlog()
        nlog.related_time = new Date(Date.now())
        return nlog
    })())
    let next_row_key = 0
    function new_row(): AddNlogRowInput {
        return { row_key: next_row_key++, title: "", amount: 0 }
    }
    // 既定は1行。「追加」で行を増やして、同じ店で買った物を一度に複数件書ける
    const nlog_rows: Ref<Array<AddNlogRowInput>> = ref([new_row()])
    const nlog_shop_value: Ref<string> = ref("")

    const related_date_typed: Ref<Date> = ref(moment().toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment().format("HH:mm:ss"))

    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Computed ──
    // 最後の1行は消せない（0行では何も保存できない）
    const can_delete_row = computed(() => nlog_rows.value.length > 1)

    // ── Business logic ──
    function add_row(): void {
        nlog_rows.value.push(new_row())
    }

    function delete_row(index: number): void {
        if (!can_delete_row.value || index < 0 || nlog_rows.value.length <= index) {
            return
        }
        nlog_rows.value.splice(index, 1)
    }

    function make_error(error_code: string, message_key: string): GkillError {
        const error = new GkillError()
        error.error_code = error_code
        error.error_message = i18n.global.t(message_key)
        return error
    }

    /** 1行ぶんの入力チェック。順は1行だったころと同じ（金額 → 店名 → タイトル） */
    function validate_row(row: AddNlogRowInput): GkillError | null {
        if (row.amount === null || row.amount.toString() === "" || Number.isNaN(Number(row.amount))) {
            return make_error(GkillErrorCodes.nlog_amount_is_blank, "NLOG_AMOUNT_IS_BLANK_MESSAGE")
        }
        if (nlog_shop_value.value === "") {
            return make_error(GkillErrorCodes.nlog_shop_name_is_blank, "NLOG_SHOP_NAME_IS_BLANK_MESSAGE")
        }
        if (row.title === "") {
            return make_error(GkillErrorCodes.nlog_title_is_blank, "NLOG_TITLE_IS_BLANK_MESSAGE")
        }
        return null
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            if (!nlog.value) {
                emits('received_errors', [make_error(GkillErrorCodes.client_nlog_is_null, "CLIENT_NLOG_IS_NULL_MESSAGE")])
                return
            }

            // 日時必須入力チェック
            if (related_date_string.value === "" || related_time_string.value === "") {
                emits('received_errors', [make_error(GkillErrorCodes.nlog_related_time_is_blank, "NLOG_DATE_TIME_IS_BLANK_MESSAGE")])
                return
            }

            // 行ごとの入力チェック。1行でも不正なら何も書かない
            for (const row of nlog_rows.value) {
                const error = validate_row(row)
                if (error !== null) {
                    emits('received_errors', [error])
                    return
                }
            }

            // タグツリーに無いタグ名なら、保存する前に確認を取る
            const tag_names = kyou_tags_view.value?.get_tag_names() ?? []
            const unknown_tags = confirm_unknown_tag.collect_unknown_tags(tag_names)
            if (unknown_tags.length !== 0) {
                confirm_unknown_tag.open_confirm(unknown_tags)
                return
            }

            await execute_save(tag_names)
        } finally {
            is_requested_submit.value = false
        }
    }

    function cancel_save(): void {
        confirm_unknown_tag.close_confirm()
    }

    async function confirm_save(): Promise<void> {
        confirm_unknown_tag.close_confirm()
        try {
            is_requested_submit.value = true
            // 確認ダイアログは非モーダルなので、確認中にタグ欄を書き換えられる。取り直す
            await execute_save(kyou_tags_view.value?.get_tag_names() ?? [])
        } finally {
            is_requested_submit.value = false
        }
    }

    async function execute_save(tag_names: Array<string>): Promise<void> {
        try {
            is_requested_submit.value = true

            // 行ごとに1件の Nlog を用意する。店名と関連時刻は全行で共有（メモ帳の支出と同じ）
            const related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
            const new_nlogs = nlog_rows.value.map(row => {
                const new_nlog = nlog.value.clone()
                new_nlog.id = props.gkill_api.generate_uuid()
                new_nlog.amount = Number(row.amount)
                new_nlog.shop = nlog_shop_value.value
                new_nlog.title = row.title
                new_nlog.related_time = related_time
                new_nlog.create_app = "gkill"
                new_nlog.create_device = props.application_config.device
                new_nlog.create_time = new Date(Date.now())
                new_nlog.create_user = props.application_config.user_id
                new_nlog.update_app = "gkill"
                new_nlog.update_device = props.application_config.device
                new_nlog.update_time = new Date(Date.now())
                new_nlog.update_user = props.application_config.user_id
                return new_nlog
            })

            // 全行の本体とタグを1つの tx に積んで commit_tx で確定する。commit は1つの SQLite
            // トランザクションなので「全部書くか、何も書かないか」になる（2行目が失敗したら1行目も残らない）。
            // tx 中の add_* は応答に Kyou もタグも載せられない（一時リポジトリにしか無い）ので、実体は commit 後に引き直す
            for (const new_nlog of new_nlogs) {
                await delete_gkill_kyou_cache(new_nlog.id)
            }
            const messages = new Array<GkillMessage>()
            const added_tags = new Array<Tag>()
            const tx = await run_in_tx(props.gkill_api, async (tx_id) => {
                // 直列に積む。並列にすると失敗したときにどこまで積んだかが決まらない
                for (const new_nlog of new_nlogs) {
                    const req = new AddNlogRequest()
                    req.nlog = new_nlog
                    req.tx_id = tx_id
                    const res = await props.gkill_api.add_nlog(req)
                    if (res.errors && res.errors.length !== 0) {
                        return res.errors
                    }
                    if (res.messages && res.messages.length !== 0) {
                        messages.push(...res.messages)
                    }
                    // タグ欄のタグは全行に付ける
                    const tag_result = await add_tags_to_target(props.gkill_api, props.application_config, new_nlog.id, tag_names, tx_id)
                    added_tags.push(...tag_result.added_tags)
                    messages.push(...tag_result.messages)
                    if (tag_result.errors.length !== 0) {
                        return tag_result.errors
                    }
                }
                return []
            })
            if (!tx.committed) {
                emits('received_errors', tx.errors)
                return
            }
            if (messages.length !== 0) {
                emits('received_messages', messages)
            }
            record_added_tag_history(props.gkill_api, tag_names)

            // タグは registered_kyou より必ず先に上げる（commit 済みなので順序が崩れることはない）。
            // 先に registered_kyou を emit すると、タグで絞り込んだ列が空のタグ列を見て「一致しない」と判定し、
            // エラーも出ないまま行が現れない。
            // 同じ名前のタグは行数ぶん付くが、受け手（タグツリーの取り直し・列条件への追加）はタグ名しか
            // 見ないので、名前ごとに1回だけ上げる（行数×タグ数の取り直しを起こさない）
            const emitted_tag_names = new Set<string>()
            added_tags.forEach(added_tag => {
                if (emitted_tag_names.has(added_tag.tag)) {
                    return
                }
                emitted_tag_names.add(added_tag.tag)
                emits('registered_tag', added_tag)
            })

            // 追加した記録は列へ局所挿入されるので、リスト全体の引き直しは要求しない。
            // Kyouが1件も引けなかったときだけ、従来どおり引き直しへ落とす
            const added_kyous = await Promise.all(new_nlogs.map(new_nlog => fetch_committed_kyou(props.gkill_api, new_nlog.id)))
            let is_emitted = false
            added_kyous.forEach((added_kyou: Kyou | null) => {
                if (!added_kyou) {
                    return
                }
                is_emitted = true
                emits('registered_kyou', added_kyou)
            })
            if (!is_emitted) {
                emits('requested_reload_list')
            }
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    function reset_related_date_time(): void {
        related_date_typed.value = moment(nlog.value.related_time).toDate()
        related_time_string.value = moment(nlog.value.related_time).format("HH:mm:ss")
    }

    function now_to_related_date_time(): void {
        related_date_typed.value = moment().toDate()
        related_time_string.value = moment().format("HH:mm:ss")
    }

    function reset(): void {
        const row = new_row()
        row.title = (nlog.value ? nlog.value.title : "")
        row.amount = (nlog.value ? nlog.value.amount : 0)
        nlog_rows.value = [row]
        nlog_shop_value.value = (nlog.value ? nlog.value.shop : "")
        related_date_typed.value = (moment().toDate())
        related_time_string.value = (moment().format("HH:mm:ss"))
        kyou_tags_view.value?.reset()
    }

    // ── CRUD relay handlers ──
    const crudRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    // ── Return ──
    return {
        // Template refs
        kyou_tags_view,
        confirm_unknown_tag_dialog: confirm_unknown_tag.confirm_unknown_tag_dialog,

        // Confirm unknown tag
        unknown_tags: confirm_unknown_tag.unknown_tags,
        cancel_save,
        confirm_save,

        // State
        is_requested_submit,
        nlog,
        nlog_rows,
        nlog_shop_value,
        related_date_typed,
        related_date_string,
        related_time_string,
        show_related_date_menu,
        show_related_time_menu,

        // Computed
        can_delete_row,

        // Business logic / template handlers
        save,
        add_row,
        delete_row,
        reset_related_date_time,
        now_to_related_date_time,
        reset,

        // Event relay objects
        crudRelayHandlers,
    }
}
