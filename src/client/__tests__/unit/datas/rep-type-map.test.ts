import { describe, test, expect, vi } from 'vitest'

// Must import i18n helper before the modules under test, so i18n.global is initialised
import { i18n } from '../../helpers/setup-i18n'

// Mock the @/i18n module to use our test i18n instance
vi.mock('@/i18n', () => ({ i18n }))

import { generate_rep_type_map } from '@/classes/datas/config/rep-type-map'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
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
})
