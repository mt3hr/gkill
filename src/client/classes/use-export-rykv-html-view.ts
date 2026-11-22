'use strict'

import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import type { ExportRykvHTMLViewEmits } from '@/pages/views/export-rykv-html-view-emits'
import type { ExportRykvHTMLViewProps } from '@/pages/views/export-rykv-html-view-props'
import { ExportRykvHTMLRequest, type ExportMediaMode, type ExportOutputFormat } from '@/classes/api/req_res/export-rykv-html-request'
import { base64_to_blob } from '@/classes/file-base64'
import { save_as } from '@/classes/save-as'

/**
 * rykv の画面を HTML へ出力するダイアログの中身。
 *
 * 出力した HTML は通信をしない（地図だけ例外）。
 * 何を焼き込むかはここで選び、組み立てはサーバ（/api/export_rykv_html）が行う。
 */
export function useExportRykvHTMLView(options: {
    props: ExportRykvHTMLViewProps,
    emits: ExportRykvHTMLViewEmits,
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

    const include_dnote: Ref<boolean> = ref(true)
    const include_map: Ref<boolean> = ref(true)
    const include_sidebar: Ref<boolean> = ref(false)

    const media_mode: Ref<ExportMediaMode> = ref('thumbnail')
    const output_format: Ref<ExportOutputFormat> = ref('single')

    const is_exporting: Ref<boolean> = ref(false)

    // ── Computed ──
    const has_focused_column: ComputedRef<boolean> = computed(() =>
        props.focused_column_index >= 0 && props.focused_column_index < props.columns.length)

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

    // ── Methods ──
    function build_request(): ExportRykvHTMLRequest {
        const req = new ExportRykvHTMLRequest()
        req.title = title.value
        // 列の条件はサイドバーが生成したまま渡す（削ると出力先で列が空になる）
        req.columns = target_columns.value
        req.include_attached_tags = include_tags.value
        req.include_attached_texts = include_texts.value
        req.include_attached_notifications = include_notifications.value
        req.include_attached_timeiss = include_timeiss.value
        req.include_gps_log = include_gps_log.value
        req.include_plugin_content = include_plugin_content.value
        req.include_dnote = include_dnote.value
        req.include_map = include_map.value
        req.include_sidebar = include_sidebar.value
        req.media_mode = media_mode.value
        req.output_format = output_format.value
        return req
    }

    async function export_html(): Promise<void> {
        if (is_exporting.value) {
            return
        }
        is_exporting.value = true
        try {
            const res = await props.gkill_api.export_rykv_html(build_request())
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
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
            is_exporting.value = false
        }
    }

    function cancel(): void {
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
        output_format,
        is_exporting,

        // Computed
        has_focused_column,
        is_sidecar_selectable,
        target_columns,

        // Methods used in template
        export_html,
        cancel,

        // Exposed for tests
        build_request,
    }
}
