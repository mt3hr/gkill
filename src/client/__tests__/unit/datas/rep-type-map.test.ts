import { describe, test, expect, vi } from 'vitest'

// Must import i18n helper before the modules under test, so i18n.global is initialised
import { i18n } from '../../helpers/setup-i18n'

// Mock the @/i18n module to use our test i18n instance
vi.mock('@/i18n', () => ({ i18n }))

import { generate_rep_type_map } from '@/classes/datas/config/rep-type-map'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'
import { GkillAPI } from '@/classes/api/gkill-api'

describe('generate_rep_type_map', () => {
    test('全RepTypeのローカライズ表示名を返す (ja)', () => {
        const map = generate_rep_type_map()
        expect(map.get('Files')).toBe('ファイル')
        expect(map.get('KC')).toBe('数値')
        expect(map.get('Kmemo')).toBe('メモ')
        expect(map.get('Lantana')).toBe('気分')
        expect(map.get('Mi')).toBe('タスク')
        expect(map.get('Nlog')).toBe('支出')
        expect(map.get('ReKyou')).toBe('リポスト')
        expect(map.get('TimeIs')).toBe('打刻')
        expect(map.get('URLog')).toBe('ブックマーク')
    })
})

describe('append_not_found_rep_types', () => {
    test('未定義RepType追加時にnameをローカライズし、keyとrep_type_nameは生の値を保持する', async () => {
        const api = {
            get_all_rep_names: vi.fn().mockResolvedValue({
                errors: [],
                messages: [],
                rep_names: ['Files_device_user', 'Kmemo_device_user', 'Unknown_device_user'],
            }),
            generate_uuid: vi.fn(() => 'test-uuid'),
        } as unknown as GkillAPI
        vi.spyOn(GkillAPI, 'get_gkill_api').mockReturnValue(api)

        const config = new ApplicationConfig()
        const errors = await config.append_not_found_rep_types()
        expect(errors).toHaveLength(0)

        const children = config.rep_type_struct.children ?? []
        const files = children.find(child => child.rep_type_name === 'Files')
        expect(files?.name).toBe('ファイル')
        expect(files?.key).toBe('Files')
        const kmemo = children.find(child => child.rep_type_name === 'Kmemo')
        expect(kmemo?.name).toBe('メモ')
        expect(kmemo?.key).toBe('Kmemo')
        // マップにないRepTypeは生の値にフォールバックする
        const unknown = children.find(child => child.rep_type_name === 'Unknown')
        expect(unknown?.name).toBe('Unknown')
    })

    test('既存エントリが未カスタマイズ(name === rep_type_name)ならローカライズ名に置き換える', async () => {
        const api = {
            get_all_rep_names: vi.fn().mockResolvedValue({
                errors: [],
                messages: [],
                rep_names: ['Kmemo_device_user', 'TimeIs_device_user', 'URLog_device_user'],
            }),
            generate_uuid: vi.fn(() => 'test-uuid'),
        } as unknown as GkillAPI
        vi.spyOn(GkillAPI, 'get_gkill_api').mockReturnValue(api)

        const config = new ApplicationConfig()

        // 旧バージョンで生の値のまま保存されたエントリ
        const raw_kmemo = new RepTypeStructElementData()
        raw_kmemo.name = 'Kmemo'
        raw_kmemo.key = 'Kmemo'
        raw_kmemo.rep_type_name = 'Kmemo'

        // フォルダ配下の生エントリ(再帰処理の確認)
        const folder = new RepTypeStructElementData()
        folder.name = 'フォルダ'
        folder.rep_type_name = 'フォルダ'
        folder.is_dir = true
        const raw_timeis = new RepTypeStructElementData()
        raw_timeis.name = 'TimeIs'
        raw_timeis.key = 'TimeIs'
        raw_timeis.rep_type_name = 'TimeIs'
        folder.children = [raw_timeis]

        // ユーザーがリネーム済みのエントリ
        const customized_urlog = new RepTypeStructElementData()
        customized_urlog.name = 'おきにいり'
        customized_urlog.key = 'URLog'
        customized_urlog.rep_type_name = 'URLog'

        // プラグイン等、マップにないRepType
        const plugin_type = new RepTypeStructElementData()
        plugin_type.name = 'claude_conversation'
        plugin_type.key = 'claude_conversation'
        plugin_type.rep_type_name = 'claude_conversation'

        config.rep_type_struct.children = [raw_kmemo, folder, customized_urlog, plugin_type]

        const errors = await config.append_not_found_rep_types()
        expect(errors).toHaveLength(0)

        expect(raw_kmemo.name).toBe('メモ')
        expect(raw_timeis.name).toBe('打刻')
        expect(customized_urlog.name).toBe('おきにいり')
        expect(plugin_type.name).toBe('claude_conversation')
        expect(folder.name).toBe('フォルダ')

        // 既存エントリは重複追加されない
        const children = config.rep_type_struct.children ?? []
        expect(children.filter(child => child.rep_type_name === 'Kmemo')).toHaveLength(1)
    })
})
