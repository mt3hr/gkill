/**
 * ApplicationConfig.clone() が構造をディープコピーすることを確かめる。
 *
 * 構造編集ダイアログは cloned_application_config の構造を in-place で書き換える
 * (children.push / splice / move_struct_*)。参照コピーのままだと実体まで到達してしまい、
 * キャンセルしても編集が取り消せない。
 * 実際に kftl_template_struct と mi_board_struct が参照コピーのままになっており、
 * KFTLテンプレート構造編集の「キャンセル」は機能していなかった。
 * 次に構造を1つ足したときも同じ穴が空かないよう、6種すべてを機械的に確かめる。
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'

// i18n を先に初期化してから被テストモジュールを読む
import { i18n } from '../../helpers/setup-i18n'
vi.mock('@/i18n', () => ({ i18n }))

import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GkillAPI } from '@/classes/api/gkill-api'

/** clone がディープコピーすべき構造フィールド */
const STRUCT_FIELDS = [
    'tag_struct',
    'rep_struct',
    'device_struct',
    'rep_type_struct',
    'kftl_template_struct',
    'mi_board_struct',
] as const

type StructField = typeof STRUCT_FIELDS[number]
type StructNode = { children?: Array<unknown> | null }

function struct_of(config: ApplicationConfig, field: StructField): StructNode {
    return (config as unknown as Record<StructField, StructNode>)[field]
}

describe('ApplicationConfig.clone', () => {
    beforeEach(() => {
        vi.restoreAllMocks()
    })

    test.each(STRUCT_FIELDS)('%s は別インスタンスになる', (field) => {
        const config = new ApplicationConfig()
        const cloned = config.clone()

        expect(struct_of(cloned, field)).not.toBe(struct_of(config, field))
    })

    test.each(STRUCT_FIELDS)('%s の子を足しても元に波及しない', (field) => {
        const config = new ApplicationConfig()
        const cloned = config.clone()

        const cloned_struct = struct_of(cloned, field)
        cloned_struct.children = cloned_struct.children ?? []
        cloned_struct.children.push({ id: 'added-by-dialog' })

        const original_children = struct_of(config, field).children ?? []
        expect(original_children).toHaveLength(0)
    })
})

describe('「無ければ足す」系はツリー全体を探す', () => {
    beforeEach(() => {
        vi.restoreAllMocks()
        vi.spyOn(GkillAPI, 'get_gkill_api').mockReturnValue({
            generate_uuid: vi.fn(() => 'test-uuid'),
        } as unknown as GkillAPI)
    })

    // 直下しか見ていなかった頃は、フォルダへ移動するたびにルートへ再生成されて2個になった
    test('フォルダの中にある「すべて」を重複生成しない', async () => {
        const config = new ApplicationConfig()
        config.mi_board_struct.children = [
            {
                ...config.mi_board_struct,
                id: 'folder-1',
                is_dir: true,
                board_name: 'フォルダ',
                key: 'フォルダ',
                name: 'フォルダ',
                children: [{ ...config.mi_board_struct, id: 'all-1', board_name: 'すべて', key: 'すべて', name: 'すべて', children: null }],
            },
        ]

        await config.append_all_mi_board()

        expect(config.mi_board_struct.children).toHaveLength(1)
    })

    test('どこにも無ければルート直下に足す', async () => {
        const config = new ApplicationConfig()
        config.mi_board_struct.children = []

        await config.append_all_mi_board()

        expect(config.mi_board_struct.children).toHaveLength(1)
        expect(config.mi_board_struct.children?.[0].board_name).toBe('すべて')
    })

    test('フォルダの中にある「no tags」を重複生成しない', async () => {
        const config = new ApplicationConfig()
        config.tag_struct.children = [
            {
                ...config.tag_struct,
                id: 'folder-1',
                is_dir: true,
                tag_name: 'フォルダ',
                key: 'フォルダ',
                name: 'フォルダ',
                children: [{ ...config.tag_struct, id: 'no-tags-1', tag_name: 'no tags', key: 'no tags', name: 'no tags', children: null }],
            },
        ]

        await config.append_no_tags()

        expect(config.tag_struct.children).toHaveLength(1)
    })

    test('フォルダの中にある「なし」デバイスを重複生成しない', async () => {
        const config = new ApplicationConfig()
        config.device_struct.children = [
            {
                ...config.device_struct,
                id: 'folder-1',
                is_dir: true,
                device_name: 'フォルダ',
                key: 'フォルダ',
                name: 'フォルダ',
                children: [{ ...config.device_struct, id: 'none-1', device_name: 'なし', key: 'なし', name: 'なし', children: null }],
            },
        ]

        await config.append_no_devices()

        expect(config.device_struct.children).toHaveLength(1)
    })
})
