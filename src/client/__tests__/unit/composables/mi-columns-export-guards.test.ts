/**
 * useMiColumns の HTML 出力向けガード（enable_open_board_column / enable_board_drag_drop）のテスト。
 *
 * 出力 Mi では焼いていない板を新しく開くと空の列が出現し、
 * D&D を通すと書き込み API を呼んでしまうため、それぞれの門が確実に閉じることを固定する。
 */
import { describe, test, expect, vi } from 'vitest'

vi.mock('@/i18n', () => ({
    default: { global: { t: (key: string) => key, locale: 'ja' } },
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
vi.mock('@/router', () => ({ default: { replace: vi.fn(), push: vi.fn() } }))
vi.mock('@/classes/delete-gkill-cache', () => ({
    default: vi.fn().mockResolvedValue(undefined),
    delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
    delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
    delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('@/classes/use-dialog-history-stack', () => ({
    reset_dialog_history: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('@/classes/use-scoped-enter-for-kftl', () => ({ useScopedEnterForKFTL: vi.fn() }))
vi.mock('@/classes/use-scoped-ctrl-v-for-clipboard', () => ({ useScopedCtrlVForClipboard: vi.fn() }))
vi.mock('@/classes/kyou-reload', () => ({
    build_mi_reload_query: vi.fn((query: unknown) => query),
    new_reload_batch: vi.fn(() => 0),
    refresh_kyou: vi.fn().mockResolvedValue(null),
    refresh_kyou_in_list: vi.fn().mockResolvedValue(undefined),
}))

import '@/classes/api/gkill-api'
import { useMiColumns } from '@/classes/use-mi-columns'
import type { MiColumnsViewProps } from '@/pages/views/mi-columns-view-props'
import type { MiColumnsViewEmits } from '@/pages/views/mi-columns-view-emits'
import {
    createColumnViewMockApi,
    makeColumnQuery,
    makeColumnViewProps,
    setupColumns,
} from '../../helpers/rykv-mi-harness'

const noop_emits = (() => {}) as unknown as MiColumnsViewEmits

function createExportedMiColumns() {
    const { api } = createColumnViewMockApi()
    const update_mi = vi.fn()
    const update_mirekyou = vi.fn()
    ;(api as unknown as Record<string, unknown>).update_mi = update_mi
    ;(api as unknown as Record<string, unknown>).update_mirekyou = update_mirekyou

    const raw_props = makeColumnViewProps(api, {}, {
        enable_context_menu: false,
        enable_board_drag_drop: false,
        enable_open_board_column: false,
        enable_keyboard_shortcuts: false,
        is_readonly_mi_check: true,
        show_column_footer: false,
        show_timeis_playing_end_button: false,
        is_sidebar_available: true,
    })
    const props = raw_props as unknown as MiColumnsViewProps
    const view = useMiColumns({ props, emits: noop_emits })
    return { view, update_mi, update_mirekyou }
}

describe('useMiColumns HTML出力向けガード', () => {
    test('enable_open_board_column=false では既に開いている板へフォーカス移動でき、未開の板は増やさない', () => {
        const { view } = createExportedMiColumns()
        const q1 = makeColumnQuery('col-1')
        q1.mi_board_name = 'BoardA'
        const q2 = makeColumnQuery('col-2')
        q2.mi_board_name = 'BoardB'
        setupColumns(view, [q1, q2], [[], []])
        view.focused_column_index.value = 0

        // 開いている板にはフォーカスを移せる
        view.open_or_focus_board('BoardB')
        expect(view.focused_column_index.value).toBe(1)
        expect(view.querys.value).toHaveLength(2)

        // 未開の板を指定しても空列を増やさない
        view.open_or_focus_board('BoardNotExported')
        expect(view.querys.value).toHaveLength(2)
        expect(view.focused_column_index.value).toBe(1)
    })

    // 門は onDropBoardTask と内側の move_board_task の2枚あるが、外から呼べるのは onDropBoardTask だけ
    // （内側の門はテンプレートの配線を1つ落としたときの保険）。ここでは外側の門を固定する
    test('enable_board_drag_drop=false では onDropBoardTask が書き込み API を呼ばず、ドロップの中身も読まない', async () => {
        const { view, update_mi, update_mirekyou } = createExportedMiColumns()
        const target_query = makeColumnQuery('col-2')
        target_query.mi_board_name = 'BoardB'

        const fake_event = {
            dataTransfer: {
                getData: vi.fn((key: string) =>
                    key === 'gkill_mi'
                        ? JSON.stringify({ id: 'mi-1', board_name: 'BoardA', title: 'Task' })
                        : '',
                ),
            },
            preventDefault: vi.fn(),
            stopPropagation: vi.fn(),
        } as unknown as DragEvent

        await view.onDropBoardTask(fake_event, target_query)
        expect(update_mi).not.toHaveBeenCalled()
        expect(update_mirekyou).not.toHaveBeenCalled()
        expect(fake_event.dataTransfer!.getData).not.toHaveBeenCalled()
    })
})
