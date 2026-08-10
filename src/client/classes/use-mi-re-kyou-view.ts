import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import type { MiReKyouViewProps } from '@/pages/views/mi-re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import { UpdateMiReKyouRequest } from '@/classes/api/req_res/update-mi-re-kyou-request'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { Kyou } from '@/classes/datas/kyou'
import { get_kyou_content_text } from '@/classes/kyou-content-text'
import { is_row_height } from '@/classes/kyou-row-height'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ComponentRef } from '@/classes/component-ref'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'
import { useDeviceKind } from '@/classes/use-device-kind'

/** 一覧の行で日時を1行だけ出すときの優先順。Miの並びに合わせる */
const MI_RE_KYOU_TIME_PRIORITY = [
    { key: 'estimate_start_time', label_key: 'MI_START_DATE_TIME_TITLE' },
    { key: 'estimate_end_time', label_key: 'MI_END_DATE_TIME_TITLE' },
    { key: 'limit_time', label_key: 'MI_LIMIT_DATE_TIME_TITLE' },
] as const

export function useMiReKyouView(options: {
    props: MiReKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref<ComponentRef | null>(null)

    // ドラッグ&ドロップはPCでのみ有効にする。
    // タブレット・スマートフォンでは長押しでcontextmenuイベントを発火させるため。
    const { is_pc } = useDeviceKind()
    const effective_draggable = computed(() => is_pc.value && (props.draggable ?? false))

    // 一覧の行に収まる高さしか無いときは参照先の埋め込みをやめる
    const is_compact = computed(() => is_row_height(props.height))

    // ── State refs ──
    const is_requested_submit = ref(false)
    const target_kyou: Ref<Kyou> = ref(new Kyou())
    const is_checked_mi: Ref<boolean> = ref(props.mirekyou.is_checked)
    // 参照先の本文を1行にしたもの。取得できるまでは高さを確定させるために「取得中」を入れておく
    const target_summary: Ref<string> = ref(i18n.global.t('LOADING_MESSAGE'))
    // 参照先が見つからなかったか。終端状態として持たないと、
    // 中身の入らないKyouViewが読み込み中表示のまま止まってしまう
    const is_target_not_found: Ref<boolean> = ref(false)
    // 取得済みのtarget_id。仮想スクロールで行を使い回すときに同じ参照先を引き直さないために持つ
    let loaded_target_id = ''

    // ── Computed ──
    /** 一覧の行では日時を1行しか出せないので、先に来る1つだけを選ぶ */
    const primary_time = computed(() => {
        for (const candidate of MI_RE_KYOU_TIME_PRIORITY) {
            const time = props.mirekyou[candidate.key]
            if (time) {
                return { label_key: candidate.label_key, time: time }
            }
        }
        return null
    })

    // ── Watchers ──
    watch(() => props.kyou, () => get_target_kyou())
    watch(() => props.mirekyou, () => {
        is_checked_mi.value = props.mirekyou.is_checked
        get_target_kyou()
    })

    // ── Business logic ──
    // force: 参照先が更新された通知(requested_reload_kyou / updated_kyou)を受けたときだけ true。
    // 参照先にタグが付いてもMiReKyou側のupdate_timeも参照先のupdate_timeも動かないので、
    // 「古くなった」をローカルに判定する材料が無い。明示的に引き直すしかない
    async function get_target_kyou(force = false) {
        // target_idが空だと下の使い回しガード(初期値'')に引っかかってリクエストすら飛ばず、
        // 中身の入らないKyouViewが読み込み中表示のまま止まる。見つからなかった扱いにして終端させる
        if (props.mirekyou.target_id === '') {
            is_target_not_found.value = true
            fallback_summary()
            return
        }
        // 仮想スクロールの行使い回しでpropsだけ差し替わることがある。参照先が同じなら引き直さない。
        // ここを外すとスクロール中に行数ぶんget_kyouが飛ぶので、forceのときだけ通す
        if (!force && loaded_target_id === props.mirekyou.target_id) {
            return
        }
        loaded_target_id = props.mirekyou.target_id
        target_summary.value = i18n.global.t('LOADING_MESSAGE')
        is_target_not_found.value = false

        const requested_target_id = props.mirekyou.target_id
        const req = new GetKyouRequest()
        req.id = requested_target_id
        const res = await props.gkill_api.get_kyou(req)
        // 応答が返るまでに行が別の参照先へ使い回されていたら捨てる。
        // loaded_target_idは連続する同一idしか抑制しないので、A→B→Aで応答が入れ替わりうる
        if (loaded_target_id !== requested_target_id) {
            return
        }
        if (res.errors && res.errors.length !== 0) {
            is_target_not_found.value = true
            fallback_summary()
            // 一覧では行数ぶんスナックバーが出てしまうので、行では黙って諦める
            if (!is_compact.value) {
                emits('received_errors', res.errors)
            }
            return
        }
        if (!res.kyou_histories || res.kyou_histories.length < 1) {
            is_target_not_found.value = true
            fallback_summary()
            if (!is_compact.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.not_found_mi_rekyou_target
                error.error_message = i18n.global.t('NOT_FOUND_MI_REKYOU_TARGET_ERROR_MESSAGE')
                emits('received_errors', [error])
            }
            return
        }
        is_target_not_found.value = false
        target_kyou.value = res.kyou_histories[0]
        await update_target_summary(requested_target_id)
    }

    /**
     * 参照先の種別データまで読んでから本文を1行に落とす。
     * 書き込みの前に毎回requested_target_idを見るのは、ここでもawaitを2回挟むため。
     * 遅い要約が、行の使い回しで別の参照先になった後のtarget_summaryを上書きしてしまう
     */
    async function update_target_summary(requested_target_id: string) {
        const errors = await target_kyou.value.load_typed_datas()
        if (loaded_target_id !== requested_target_id) {
            return
        }
        if (errors && errors.length !== 0) {
            fallback_summary()
            return
        }
        // 行から呼ぶときは、プラグインのContent HTMLを取りに行かせず、
        // 参照先のさらに参照先も1段までしかたどらせない。件数ぶんリクエストが走るため
        const { text } = await get_kyou_content_text(target_kyou.value, props.gkill_api, 0, {
            allow_remote: !is_compact.value,
            max_lazy_depth: is_compact.value ? 1 : undefined,
        })
        if (loaded_target_id !== requested_target_id) {
            return
        }
        if (text.length === 0) {
            fallback_summary()
            return
        }
        target_summary.value = text
    }

    function fallback_summary() {
        target_summary.value = i18n.global.t('MI_REKYOU_TITLE')
    }

    function show_context_menu(e: PointerEvent): void {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    // 一覧上のチェックはそのままサーバ更新に繋がる。連打で同じ更新が重なるのを防ぐ
    async function clicked_mi_check(): Promise<void> {
        // 読み取り専用表示だったら何もしない
        if (props.is_readonly_mi_check) {
            return
        }
        if (is_requested_submit.value) {
            return
        }
        is_requested_submit.value = true
        try {
            await update_mi_check()
        } finally {
            is_requested_submit.value = false
        }
    }

    async function update_mi_check(): Promise<void> {
        is_checked_mi.value = !is_checked_mi.value

        // 更新がなかったらエラーメッセージを出力する
        if (props.mirekyou.is_checked === is_checked_mi.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.mi_rekyou_is_no_update
            error.error_message = i18n.global.t("MI_REKYOU_IS_NO_UPDATE_MESSAGE")
            emits('received_errors', [error])
            return
        }

        // 更新後mirekyou情報を用意する
        const updated_mirekyou = props.mirekyou.clone()
        updated_mirekyou.is_checked = is_checked_mi.value
        updated_mirekyou.update_app = "gkill"
        updated_mirekyou.update_device = props.application_config.device
        updated_mirekyou.update_time = new Date(Date.now())
        updated_mirekyou.update_user = props.application_config.user_id

        // 更新リクエストを飛ばす
        await delete_gkill_kyou_cache(updated_mirekyou.id)
        const req = new UpdateMiReKyouRequest()
        req.mirekyou = updated_mirekyou
        req.want_response_kyou = true

        const res = await props.gkill_api.update_mirekyou(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits('requested_reload_kyou', props.kyou)
        return
    }

    function onDragStart(e: DragEvent) {
        e.dataTransfer!.setData("gkill_mi_re_kyou", JSON.stringify(props.mirekyou))
    }

    // ── Init ──
    get_target_kyou()

    // ── Event relay objects ──
    // 参照先のタグ等が変わっても、このMiReKyou行が抱えている target_kyou は
    // 使い回しガードのせいで引き直されない。対象idが一致したときだけ強制的に引き直す
    const crudRelayHandlers = build_kyou_view_relay(emits, {
        'requested_reload_kyou': (kyou: Kyou) => {
            if (kyou.id === props.mirekyou.target_id) {
                get_target_kyou(true)
            }
            emits('requested_reload_kyou', kyou)
        },
        'updated_kyou': (kyou: Kyou) => {
            if (kyou.id === props.mirekyou.target_id) {
                get_target_kyou(true)
            }
            emits('updated_kyou', kyou)
        },
    })

    // ── Return ──
    return {
        // Template refs
        context_menu,

        // State
        target_kyou,
        is_requested_submit,
        is_checked_mi,
        target_summary,
        is_target_not_found,
        effective_draggable,

        // Computed
        is_compact,
        primary_time,

        // Business logic
        show_context_menu,
        get_target_kyou,
        clicked_mi_check,
        onDragStart,

        // Event relay objects
        crudRelayHandlers,
    }
}

