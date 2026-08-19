/**
 * 検索条件エディタダイアログの初期値規則の検証。
 *
 * 規則:「値がセットされていれば(query_idが空でなければ)それを優先し、
 * 無ければエディタが ApplicationConfig のデフォルト検索条件を適用する」。
 * query_id の空文字は「値が未セット」の印としてエディタ側の判定に使われるため、
 * ダイアログの show() が無条件に採番すると印が潰れて既定が一切効かなくなる
 * (2026-08-10 の「検索条件ダイアログにデフォルトが適用されない」回帰)。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('@/classes/use-dialog-history-stack', () => ({
    useDialogHistoryStack: vi.fn(),
    close_dialog_via_history: vi.fn(),
}))
vi.mock('@/classes/use-floating-dialog', () => ({
    useFloatingDialog: vi.fn(() => ({})),
}))

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

import { ref } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { useFindQueryEditorDialog } from '@/classes/use-find-query-editor-dialog'
import { useMiFindQueryEditorDialog } from '@/classes/use-mi-find-query-editor-dialog'
import { useFindTimeIsQueryEditorDialog } from '@/classes/use-find-time-is-query-editor-dialog'

function make_options() {
    const props = {
        gkill_api: { generate_uuid: () => 'renewed-uuid' },
        application_config: {},
    }
    return {
        props,
        emits: (() => { }),
        model_value: ref<FindKyouQuery | undefined>(undefined),
    }
}

async function flush(times = 4): Promise<void> {
    for (let i = 0; i < times; i++) {
        await Promise.resolve()
    }
}

const dialogs = [
    { name: 'FindQueryEditorDialog', use: useFindQueryEditorDialog },
    { name: 'MiFindQueryEditorDialog', use: useMiFindQueryEditorDialog },
    { name: 'FindTimeIsQueryEditorDialog', use: useFindTimeIsQueryEditorDialog },
] as const

describe.each(dialogs)('$name の show()', ({ use }) => {
    test('query_idが空(値が未セット)ならそのまま渡し、エディタの既定適用判定を潰さない', async () => {
        const view = use(make_options() as unknown as Parameters<typeof use>[0])
        await view.show(new FindKyouQuery())
        await flush()

        expect(view.cloned_find_kyou_query.value?.query_id).toBe('')
    })

    test('セット済みのクエリは呼び出し元とID衝突しないよう新IDへ振り直す(既存値優先)', async () => {
        const view = use(make_options() as unknown as Parameters<typeof use>[0])
        const stored = new FindKyouQuery()
        stored.query_id = 'stored-id'
        stored.keywords = '保存済みの条件'
        await view.show(stored)
        await flush()

        expect(view.cloned_find_kyou_query.value?.query_id).toBe('renewed-uuid')
        expect(view.cloned_find_kyou_query.value?.keywords, '既存値が失われている').toBe('保存済みの条件')
    })
})
