import { describe, test, expect, vi, beforeEach } from 'vitest'

// Must import i18n helper before the modules under test, so i18n.global is initialised
import { i18n } from '../../helpers/setup-i18n'

// Mock the @/i18n module to use our test i18n instance
vi.mock('@/i18n', () => ({ i18n }))

import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GkillAPI } from '@/classes/api/gkill-api'

function tag_names_in(config: ApplicationConfig): Array<string> {
    const names = new Array<string>()
    const walk = (element: { tag_name?: string, children?: Array<unknown> }): void => {
        if (element.tag_name) {
            names.push(element.tag_name)
        }
        for (const child of (element.children ?? [])) {
            if (child) {
                walk(child as { tag_name?: string, children?: Array<unknown> })
            }
        }
    }
    walk(config.tag_struct)
    return names
}

describe('append_not_found_tags', () => {
    beforeEach(() => {
        vi.restoreAllMocks()
    })

    test('追加するタグがあるときはforce_regetで問い直し、その結果のみをTagStructに追加する', async () => {
        // 1回目はServiceWorkerがキャッシュした古い一覧 (編集前のタグ名'お避け'を含む)
        // 2回目はforce_regetでサーバから取り直した最新の一覧
        const get_all_tag_names = vi.fn()
            .mockResolvedValueOnce({ errors: [], messages: [], tag_names: ['お避け', 'gkill'] })
            .mockResolvedValueOnce({ errors: [], messages: [], tag_names: ['お酒', 'gkill'] })
        const api = {
            get_all_tag_names,
            generate_uuid: vi.fn(() => 'test-uuid'),
        } as unknown as GkillAPI
        vi.spyOn(GkillAPI, 'get_gkill_api').mockReturnValue(api)

        const config = new ApplicationConfig()
        const errors = await config.append_not_found_tags()
        expect(errors).toHaveLength(0)

        expect(get_all_tag_names).toHaveBeenCalledTimes(2)
        expect(get_all_tag_names.mock.calls[0][0].force_reget).toBe(false)
        expect(get_all_tag_names.mock.calls[1][0].force_reget).toBe(true)

        const names = tag_names_in(config)
        expect(names).toContain('お酒')
        expect(names).toContain('gkill')
        // 編集前のタグ名は追加されない
        expect(names).not.toContain('お避け')
    })

    test('追加するタグがないときは問い直さない', async () => {
        const get_all_tag_names = vi.fn()
            .mockResolvedValue({ errors: [], messages: [], tag_names: [] })
        const api = {
            get_all_tag_names,
            generate_uuid: vi.fn(() => 'test-uuid'),
        } as unknown as GkillAPI
        vi.spyOn(GkillAPI, 'get_gkill_api').mockReturnValue(api)

        const config = new ApplicationConfig()
        const errors = await config.append_not_found_tags()
        expect(errors).toHaveLength(0)
        expect(get_all_tag_names).toHaveBeenCalledTimes(1)
    })
})
