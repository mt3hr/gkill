import { describe, test, expect, vi, beforeEach } from 'vitest'

// Must import i18n helper before the modules under test, so i18n.global is initialised
import { i18n } from '../../helpers/setup-i18n'

// Mock the @/i18n module to use our test i18n instance
vi.mock('@/i18n', () => ({ i18n }))

import { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import { GkillAPI } from '@/classes/api/gkill-api'

// rep_name を持たない異常ノード(REP_TYPE_STRUCT の内容が REP_STRUCT キーへ
// 保存されていた実DBの実例)。ここで throw すると設定読み込みが丸ごと死ぬ
function foreign_rep_type_node(name: string): RepStructElementData {
    return {
        name,
        seq_in_parent: 0,
        id: `foreign-${name}`,
        rep_type_name: name,
        check_when_inited: true,
        children: null,
        key: name,
        is_checked: true,
        indeterminate: false,
        is_dir: false,
        is_open_default: false,
    } as unknown as RepStructElementData
}

function valid_rep_node(rep_name: string): RepStructElementData {
    return {
        name: rep_name,
        id: `rep-${rep_name}`,
        rep_name,
        check_when_inited: true,
        ignore_check_rep_rykv: false,
        children: null,
        key: rep_name,
        is_checked: true,
        indeterminate: false,
        is_dir: false,
    } as unknown as RepStructElementData
}

function rep_names_in(config: ApplicationConfig): Array<string> {
    const names = new Array<string>()
    const walk = (element: { rep_name?: string, children?: Array<unknown> | null }): void => {
        for (const child of (element.children ?? [])) {
            if (child) {
                const rep_name = (child as { rep_name?: string }).rep_name
                if (typeof rep_name === 'string') {
                    names.push(rep_name)
                }
                walk(child as { rep_name?: string, children?: Array<unknown> | null })
            }
        }
    }
    walk(config.rep_struct)
    return names
}

describe('append_not_found_reps', () => {
    beforeEach(() => {
        vi.restoreAllMocks()
        const api = {
            generate_uuid: vi.fn(() => 'test-uuid'),
        } as unknown as GkillAPI
        vi.spyOn(GkillAPI, 'get_gkill_api').mockReturnValue(api)
    })

    test('ツリーにない実repは末尾へ追加され、既にあるrepは重複しない', async () => {
        const config = new ApplicationConfig()
        config.rep_struct.children = [valid_rep_node('kmemo_laptop_2024')]

        const errors = await config.append_not_found_reps(['kmemo_laptop_2024', 'urlog_laptop_2024'])
        expect(errors).toHaveLength(0)

        expect(rep_names_in(config)).toEqual(['kmemo_laptop_2024', 'urlog_laptop_2024'])
    })

    test('rep_name を持たない異常ノードが混ざっていても追加処理は落ちない', async () => {
        const config = new ApplicationConfig()
        config.rep_struct.children = [
            foreign_rep_type_node('Box'),
            valid_rep_node('kmemo_laptop_2024'),
        ]

        const errors = await config.append_not_found_reps(['kmemo_laptop_2024', 'urlog_laptop_2024'])
        expect(errors).toHaveLength(0)

        expect(rep_names_in(config)).toEqual(['kmemo_laptop_2024', 'urlog_laptop_2024'])
    })
})
