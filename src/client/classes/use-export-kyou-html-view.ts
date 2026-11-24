'use strict'

import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import type { ExportKyouHTMLViewEmits } from '@/pages/views/export-kyou-html-view-emits'
import type { ExportKyouHTMLViewProps } from '@/pages/views/export-kyou-html-view-props'
import {
    ExportKyouHTMLRequest,
    EXPORT_DEFAULT_MEDIA_BUDGET_BYTES,
    EXPORT_DEFAULT_MEDIA_ITEM_LIMIT_BYTES,
    type ExportMediaMode,
    type ExportOutputFormat,
} from '@/classes/api/req_res/export-kyou-html-request'
import { base64_to_blob } from '@/classes/file-base64'
import { save_as } from '@/classes/save-as'

/**
 * 画面を HTML へ出力するダイアログの中身。
 *
 * 出力した HTML は通信をしない（地図だけ例外）。
 * 何を焼き込むかはここで選び、組み立てはサーバ（/api/export_kyou_html）が行う。
 */
export function useExportKyouHTMLView(options: {
    props: ExportKyouHTMLViewProps,
    emits: ExportKyouHTMLViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const title: Ref<string> = ref('')
    /** 'all' = 表示中のすべての列 / 'focused' = フォーカスされている列 */
    const target: Ref<'all' | 'focused'> = ref('all')

    const include_tags: Ref<boolean> = ref(true)
    const include_texts: Ref<boolean> = ref(true)
    const include_notifications: Ref<boolean> = ref(true)
    const include_timeiss: Ref<boolean> = ref(true)
    const include_gps_log: Ref<boolean> = ref(true)
    const include_plugin_content: Ref<boolean> = ref(false)

    // 「画面」の節の集計（Dnote）とマップは rykv にしか無い。選べるかは screen から決め、
    // 別のフラグで渡さない（screen と食い違う組み合わせを作れてしまう。ADR-0418）。
    // **選べないときは false 固定で送る** —— 選べないものを要求に載せると、焼き込みの印と中身が食い違う
    const can_choose_dnote: ComputedRef<boolean> = computed(() => props.screen === 'rykv')
    const can_choose_map: ComputedRef<boolean> = computed(() => props.screen === 'rykv')

    // 選べない画面（mi の集計・マップ）は最初から false。
    // チェック欄を disabled にするだけだと、選択済みの値が送信まで残る
    const include_dnote: Ref<boolean> = ref(can_choose_dnote.value)
    const include_map: Ref<boolean> = ref(can_choose_map.value)
    const include_sidebar: Ref<boolean> = ref(false)

    const media_mode: Ref<ExportMediaMode> = ref('thumbnail')
    // 予算は MB で持つ（入力欄がそのまま使う）。送るときにバイトへ直す
    const media_budget_mb: Ref<number> = ref(EXPORT_DEFAULT_MEDIA_BUDGET_BYTES / 1024 / 1024)
    const media_item_limit_mb: Ref<number> = ref(EXPORT_DEFAULT_MEDIA_ITEM_LIMIT_BYTES / 1024 / 1024)
    const output_format: Ref<ExportOutputFormat> = ref('single')

    const is_exporting: Ref<boolean> = ref(false)
    // 出力中の要求。キャンセルで中断するために握っておく
    let abort_controller: AbortController | null = null

    // ── Computed ──
    const has_focused_column: ComputedRef<boolean> = computed(() =>
        props.focused_column_index >= 0 && props.focused_column_index < props.columns.length)

    // 「対象」の文言は画面で変わる（rykv は列、mi は板）。
    // i18n キーを返すだけにして、テンプレート側で t() を通す
    const target_all_label_key: ComputedRef<string> = computed(() =>
        props.screen === 'mi' ? 'EXPORT_TARGET_ALL_BOARDS' : 'EXPORT_TARGET_ALL_COLUMNS')
    const target_focused_label_key: ComputedRef<string> = computed(() =>
        props.screen === 'mi' ? 'EXPORT_TARGET_FOCUSED_BOARD' : 'EXPORT_TARGET_FOCUSED_COLUMN')

    /** メディアを埋め込まないと同名フォルダは空になるので選べない */
    const is_sidecar_selectable: ComputedRef<boolean> = computed(() => media_mode.value !== 'none')

    const target_columns: ComputedRef<Array<typeof props.columns[number]>> = computed(() => {
        if (target.value === 'focused' && has_focused_column.value) {
            return [props.columns[props.focused_column_index]]
        }
        return props.columns
    })

    // ── Watchers ──
    // **disabled にするだけでは足りない。** 先に同名フォルダを選んでから
    // 「埋め込まない」へ変えると、選択済みの 'sidecar' が送信まで残り、
    // 中身の無いフォルダを作ろうとしてサーバに弾かれる。ref も戻す
    watch(media_mode, () => {
        if (media_mode.value === 'none' && output_format.value === 'sidecar') {
            output_format.value = 'single'
        }
    })

    // 出力中かをダイアログへ伝える。ダイアログはこれが立っている間、閉じない
    watch(is_exporting, () => emits('changed_is_exporting', is_exporting.value))

    // ── Methods ──
    /** MB の入力をバイトへ。空・NaN・0 以下は既定へ倒す */
    function to_budget_bytes(mb: number, fallback_bytes: number): number {
        if (!Number.isFinite(mb) || mb <= 0) {
            return fallback_bytes
        }
        return Math.floor(mb * 1024 * 1024)
    }

    function build_request(): ExportKyouHTMLRequest {
        const req = new ExportKyouHTMLRequest()
        req.title = title.value
        // 列の条件はサイドバーが生成したまま渡す（削ると出力先で列が空になる）
        req.columns = target_columns.value
        req.include_attached_tags = include_tags.value
        req.include_attached_texts = include_texts.value
        req.include_attached_notifications = include_notifications.value
        req.include_attached_timeiss = include_timeiss.value
        req.include_gps_log = include_gps_log.value
        req.include_plugin_content = include_plugin_content.value
        // 焼き込みの印。出力先のページがこれを見てマウント先を選ぶ
        req.screen = props.screen
        // 選べない画面は false 固定で送る。ref をそのまま送ると、
        // props が後から変わったときに「選べないのに true」が漏れる
        req.include_dnote = can_choose_dnote.value && include_dnote.value
        req.include_map = can_choose_map.value && include_map.value
        req.include_sidebar = include_sidebar.value
        req.media_mode = media_mode.value
        // 入力が空や 0 以下なら既定に任せる（サーバ側も 0 以下を既定へ倒す）
        req.media_budget_bytes = to_budget_bytes(media_budget_mb.value, EXPORT_DEFAULT_MEDIA_BUDGET_BYTES)
        req.media_item_limit_bytes = to_budget_bytes(media_item_limit_mb.value, EXPORT_DEFAULT_MEDIA_ITEM_LIMIT_BYTES)
        req.output_format = output_format.value
        return req
    }

    async function export_html(): Promise<void> {
        if (is_exporting.value) {
            return
        }
        is_exporting.value = true
        const req = build_request()
        abort_controller = req.abort_controller
        try {
            const res = await props.gkill_api.export_kyou_html(req)
            // 中断されていたら、返ってきたものは捨てる（保存もしないし閉じもしない）
            if (!is_exporting.value) {
                return
            }
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            // **警告は必ず出す。** 予算に収まらなかったメディアや読めなかったファイルは
            // ここでしか伝わらず、黙っていると「画像が出ない」としか見えない。
            // サーバが成功の知らせと並べて warning のメッセージ（MSG000096）で返すので、
            // そのまま流せばフィードが閉じるまで残す
            const messages = res.messages ?? []
            if (messages.length !== 0) {
                emits('received_messages', messages)
            }
            // 応答は JSON。ファイルそのものは body に載せられない（ADR-0712）ので、
            // ここで Blob へ戻して保存する
            if (res.output_format === 'sidecar') {
                save_as(base64_to_blob(res.zip_base64, 'application/zip'), res.file_name)
            } else {
                save_as(new Blob([res.html], { type: 'text/html' }), res.file_name)
            }
            emits('requested_close_dialog')
        } finally {
            abort_controller = null
            is_exporting.value = false
        }
    }

    /**
     * キャンセル。**出力中は閉じずに中断だけする。**
     *
     * 出力の最中にダイアログを閉じられると、応答が返ってきたときに保存の相手が居ない。
     * 中断してから、もう一度押せば閉じる（閉じてよいかは is_exporting が決める。
     * ダイアログ側の × と Esc も同じ条件で止めてある）。
     */
    function cancel(): void {
        if (is_exporting.value) {
            abort_controller?.abort()
            abort_controller = null
            is_exporting.value = false
            return
        }
        emits('requested_close_dialog')
    }

    // ── Return ──
    return {
        // State
        title,
        target,
        include_tags,
        include_texts,
        include_notifications,
        include_timeiss,
        include_gps_log,
        include_plugin_content,
        include_dnote,
        include_map,
        include_sidebar,
        media_mode,
        media_budget_mb,
        media_item_limit_mb,
        output_format,
        is_exporting,

        // Computed
        can_choose_dnote,
        can_choose_map,
        has_focused_column,
        target_all_label_key,
        target_focused_label_key,
        is_sidecar_selectable,
        target_columns,

        // Methods used in template
        export_html,
        cancel,

        // Exposed for tests
        build_request,
    }
}
