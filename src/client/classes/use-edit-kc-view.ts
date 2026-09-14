import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { EditKCViewProps } from '@/pages/views/edit-kc-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateKCRequest } from '@/classes/api/req_res/update-kc-request'
import moment from 'moment'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'
import type { ComponentRef } from '@/classes/component-ref'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { apply_kyou_tag_changes, record_added_tag_history, type ApplyTagChangesResult } from '@/classes/kyou-tags'
import { fetch_committed_kyou, run_in_tx } from '@/classes/gkill-tx'
import type { GkillMessage } from '@/classes/api/gkill-message'

export function useEditKCView(options: {
    props: EditKCViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_tags_view = ref<ComponentRef | null>(null)

    // ── Confirm unknown tag ──
    const confirm_unknown_tag = useConfirmUnknownTag({ application_config: () => props.application_config })

    // ── State refs ──
    const is_loading = ref(true)
    const is_requested_submit = ref(false)
    const is_busy = computed(() => is_loading.value || is_requested_submit.value)
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const title: Ref<string> = ref(cloned_kyou.value.typed_kc ? cloned_kyou.value.typed_kc.title : "")
    const num_value: Ref<number> = ref(cloned_kyou.value.typed_kc ? cloned_kyou.value.typed_kc.num_value : 0)
    const related_date_typed: Ref<Date> = ref(moment(cloned_kyou.value.related_time).toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(cloned_kyou.value.related_time).format("HH:mm:ss"))
    const show_kyou: Ref<boolean> = ref(false)
    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Business logic ──
    async function load(): Promise<void> {
        try {
            is_loading.value = true
            cloned_kyou.value = props.kyou.clone()
            await cloned_kyou.value.load_typed_datas()
            title.value = cloned_kyou.value.typed_kc ? cloned_kyou.value.typed_kc.title : ""
            num_value.value = cloned_kyou.value.typed_kc ? cloned_kyou.value.typed_kc.num_value : 0
            related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
            related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
        } finally {
            is_loading.value = false
        }
    }

    /**
     * KCの中身か関連日時が変わっているか。
     *
     * タグだけを足したときにこれが偽なら update_kc は呼ばない。
     * 呼ぶと中身の同じ新しい版が1つ増えてしまう
     */
    function is_body_changed(): boolean {
        const kc = cloned_kyou.value.typed_kc
        if (!kc) {
            return false
        }
        return !(kc.title === title.value && kc.num_value === num_value.value &&
            moment(kc.related_time).toDate().getTime() === moment(related_date_string.value + " " + related_time_string.value).toDate().getTime())
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            cloned_kyou.value.abort_controller.abort()
            cloned_kyou.value.abort_controller = new AbortController()

            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            const kc = cloned_kyou.value.typed_kc
            if (!kc) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_kc_is_null
                error.error_message = i18n.global.t("CLIENT_KC_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // タイトル入力チェック
            if (title.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.kc_title_is_blank
                error.error_message = i18n.global.t("KC_TITLE_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 数値入力チェック
            if (num_value.value === null) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.kc_num_value_is_blank
                error.error_message = i18n.global.t("KC_NUM_VALUE_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 日時必須入力チェック
            if (related_date_string.value === "" || related_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.kc_related_time_is_blank
                error.error_message = i18n.global.t("KC_RELATED_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新がなかったらエラーメッセージを出力する。
            // 関連日時もこの画面で編集できるので比較に含める（含めないと日時だけ変えても保存できない）。
            // タグだけを足した/外したときもここで弾かれないよう、タグの変更も更新とみなす
            const has_tag_changes = kyou_tags_view.value?.has_pending_changes() ?? false
            if (!is_body_changed() && !has_tag_changes) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.kc_is_no_update
                error.error_message = i18n.global.t("KC_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // タグツリーに無いタグ名なら、保存する前に確認を取る
            const tag_names = kyou_tags_view.value?.get_tag_names() ?? []
            const unknown_tags = confirm_unknown_tag.collect_unknown_tags(tag_names)
            if (unknown_tags.length !== 0) {
                confirm_unknown_tag.open_confirm(unknown_tags)
                return
            }

            await execute_save()
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
            await execute_save()
        } finally {
            is_requested_submit.value = false
        }
    }

    async function execute_save(): Promise<void> {
        try {
            is_requested_submit.value = true
            const kc = cloned_kyou.value.typed_kc
            if (!kc) {
                return
            }

            // 本体の更新とタグの変更を1つの tx に積んで commit_tx で確定する（全部書くか、何も書かないか）。
            // tx 中の update_* は応答に Kyou もタグも載せられないので、実体は commit 後に引き直す
            const body_changed = is_body_changed()
            const messages = new Array<GkillMessage>()
            // クロージャの中で代入するので、TS の制御フロー解析に潰されないよう入れ物に持つ
            const staged: { tag_changes: ApplyTagChangesResult | null } = { tag_changes: null }
            const tx = await run_in_tx(props.gkill_api, async (tx_id) => {
                // 中身が変わったときだけ更新リクエストを積む（変わっていないのに積むと中身の同じ新しい版が1つ増える）
                if (body_changed) {
                    const updated_kc = kc.clone()
                    updated_kc.title = title.value
                    updated_kc.num_value = num_value.value
                    updated_kc.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
                    updated_kc.update_app = "gkill"
                    updated_kc.update_device = props.application_config.device
                    updated_kc.update_time = new Date(Date.now())
                    updated_kc.update_user = props.application_config.user_id

                    await delete_gkill_kyou_cache(updated_kc.id)
                    const req = new UpdateKCRequest()
                    req.kc = updated_kc
                    req.tx_id = tx_id
                    const res = await props.gkill_api.update_kc(req)
                    if (res.errors && res.errors.length !== 0) {
                        return res.errors
                    }
                    if (res.messages && res.messages.length !== 0) {
                        messages.push(...res.messages)
                    }
                }

                // 確認ダイアログは非モーダルなので、確認中にタグ欄を書き換えられる。取り直す
                staged.tag_changes = await stage_tag_changes(tx_id)
                messages.push(...staged.tag_changes.messages)
                return staged.tag_changes.errors
            })
            if (!tx.committed) {
                emits('received_errors', tx.errors)
                return
            }
            if (messages.length !== 0) {
                emits('received_messages', messages)
            }
            const committed_tag_changes = staged.tag_changes
            if (committed_tag_changes) {
                record_added_tag_history(props.gkill_api, committed_tag_changes.added_tags.map(added_tag => added_tag.tag))
            }

            if (body_changed) {
                const updated_kyou = await fetch_committed_kyou(props.gkill_api, props.kyou.id)
                if (updated_kyou) {
                    emits('updated_kyou', updated_kyou)
                }
            }
            committed_tag_changes?.added_tags.forEach(added_tag => emits('registered_tag', added_tag))
            committed_tag_changes?.removed_tags.forEach(removed_tag => emits('deleted_tag', removed_tag))

            // タグの変更は updated_kyou を出さないので、これが唯一の反映信号になる
            emits('requested_reload_kyou', props.kyou)
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    /** タグ欄で足したもの・外したものを tx に積む。emit は呼び出し元が commit 後に行う */
    async function stage_tag_changes(tx_id: string): Promise<ApplyTagChangesResult> {
        const tags_view = kyou_tags_view.value
        if (!tags_view) {
            return { added_tags: [], removed_tags: [], errors: [], messages: [] }
        }
        return apply_kyou_tag_changes(props.gkill_api, props.application_config, cloned_kyou.value.id,
            tags_view.get_tag_names(), tags_view.get_removed_tags(), tx_id)
    }

    function now_to_related_date_time(): void {
        related_date_typed.value = moment().toDate()
        related_time_string.value = moment().format("HH:mm:ss")
    }

    function reset_related_date_time(): void {
        related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
        related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
    }

    function reset(): void {
        title.value = cloned_kyou.value.typed_kc ? cloned_kyou.value.typed_kc.title : ""
        related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
        related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
        kyou_tags_view.value?.reset()
    }

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Init ──
    load()

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
        is_loading,
        is_requested_submit,
        is_busy,
        cloned_kyou,
        title,
        num_value,
        related_date_typed,
        related_date_string,
        related_time_string,
        show_kyou,
        show_related_date_menu,
        show_related_time_menu,

        // Methods
        save,
        now_to_related_date_time,
        reset_related_date_time,
        reset,

        // Event relay
        crudRelayHandlers,
    }
}

