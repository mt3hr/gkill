'use strict'

import { gkill_page_list } from '@/classes/gkill-page-list'
import router from '@/router'
import { ref } from 'vue'
import type { MiViewEmits } from '@/pages/views/mi-view-emits'
import type { MiViewProps } from '@/pages/views/mi-view-props'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import type { ComponentRef } from '@/classes/component-ref'
import { build_mi_columns_forward_relay } from '@/pages/views/mi-columns-view-emits'

/**
 * 普段使う mi（タスク画面）のラッパ。
 *
 * **列の機構は持たない**（`use-mi-columns.ts`）。ここが持つのは
 * 「編集できる mi にだけあるもの」—— アプリバーの画面切替メニュー、
 * 編集系のダイアログ群、FAB、ヘルプ。HTML 出力のラッパ（`exported-mi-view.vue`）は
 * これらを1つも持たないので、同じ列の子を別のラッパがホストする形にしてある。
 *
 * rykv の対になる実装は `use-rykv-view.ts`（**修正は必ず両方へ**）。
 * 経緯と却下案は documents/adr/0420-split-mi-views-by-mode.md。
 */
export function useMiView(options: {
    props: MiViewProps,
    emits: MiViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
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
    const export_kyou_html_dialog = ref<ComponentRef | null>(null)
    const help_dialog = ref<ComponentRef | null>(null)

    // 画面切替メニューの一覧は classes/gkill-page-list.ts に1つだけ置いてある
    const page_list = gkill_page_list

    // ── Template event handlers ──
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

    function show_export_kyou_html_dialog(): void {
        export_kyou_html_dialog.value?.show()
    }

    function show_help_dialog(): void {
        help_dialog.value?.show()
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
    // 列が上げてくる CRUD をページへ素通しする束。**名前の網羅は型で縛られている**
    // （`mi-columns-view-emits.ts`）。手書きで並べると1件落ちても型では落ちず、
    // 「編集したのにタグツリーが更新されない」のような形で静かに壊れる
    const columnsForwardHandlers = build_mi_columns_forward_relay(emits)

    // ── Return ──
    return {
        // Template refs
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
        export_kyou_html_dialog,
        help_dialog,

        // Computed
        page_list,

        // Template event handlers
        navigate_to_page,

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
        show_export_kyou_html_dialog,
        show_help_dialog,
        floating_action_button_style,

        // Event relay objects
        columnsForwardHandlers,
    }
}
