/**
 * get_kyou_content_text のテスト。
 * 種別ごとの本文の取り出しと、参照先をたどる種別 (ReKyou / MiReKyou) の挙動を検証する。
 */
import { describe, test, expect, vi } from 'vitest'
// kyou-content-text は req_res 経由で GkillAPIRequest に依存する。
// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させないと class extends が undefined になる。
import '@/classes/api/gkill-api'
import { get_kyou_content_text } from '@/classes/kyou-content-text'
import type { GkillAPI } from '@/classes/api/gkill-api'
import type { Kyou } from '@/classes/datas/kyou'
import {
    makeKyou,
    makeKyouWithKmemo,
    makeKyouWithKc,
    makeKyouWithMi,
    makeKyouWithNlog,
    makeKyouWithLantana,
    makeKyouWithTimeis,
    makeKyouWithURLog,
    makeKyouWithGitCommitLog,
    makeMiReKyou,
    makeReKyou,
} from '../../helpers/factory'

function asKyou(kyou: unknown): Kyou {
    return kyou as Kyou
}

type MockApi = GkillAPI & {
    get_plugin_content_html: ReturnType<typeof vi.fn>
    get_kyou: ReturnType<typeof vi.fn>
}

function makeApi(html = '', target_kyou: unknown = null): MockApi {
    const api = {
        get_plugin_content_html: vi.fn().mockResolvedValue({ errors: [], html: html }),
        // 参照先の遅延解決で使う。既定は「見つからない」
        get_kyou: vi.fn().mockResolvedValue({
            errors: [],
            kyou_histories: target_kyou === null ? [] : [target_kyou],
        }),
    }
    return api as unknown as MockApi
}

describe('get_kyou_content_text 種別ごとの本文', () => {
    // MiReKyou分岐を足すにあたって、他種別の既存の出力が変わっていないことを固定する
    const cases: Array<{ name: string, kyou: unknown, expected: string }> = [
        { name: 'kmemo は content', kyou: makeKyouWithKmemo('メモ本文'), expected: 'メモ本文' },
        { name: 'kc は title と num_value', kyou: makeKyouWithKc('体重', 62), expected: '体重 62' },
        { name: 'urlog は url', kyou: makeKyouWithURLog('Example', 'https://example.com'), expected: 'https://example.com' },
        { name: 'nlog は shop / title / amount', kyou: makeKyouWithNlog('店', '買い物', 1200), expected: '店 買い物 1200' },
        { name: 'timeis は title', kyou: makeKyouWithTimeis('作業'), expected: '作業' },
        { name: 'mi は title', kyou: makeKyouWithMi('タスク'), expected: 'タスク' },
        { name: 'lantana は mood', kyou: makeKyouWithLantana(5), expected: '5' },
        { name: 'git_commit_log は commit_message', kyou: makeKyouWithGitCommitLog('fix: 直した'), expected: 'fix: 直した' },
        {
            name: 'idf_kyou は file_name',
            kyou: makeKyou({ data_type: 'idf', typed_idf_kyou: { file_name: 'photo.png' } }),
            expected: 'photo.png',
        },
        { name: 'どの種別にも該当しなければ空文字', kyou: makeKyou(), expected: '' },
    ]

    test.each(cases)('$name', async ({ kyou, expected }) => {
        const { text, errors } = await get_kyou_content_text(asKyou(kyou), makeApi())
        expect(errors).toEqual([])
        expect(text).toBe(expected)
    })

    test('kmemo の前後の空白と余分な改行は落とす', async () => {
        const kyou = makeKyouWithKmemo('  1行目  \r\n\n\n\n  2行目  \n')
        const { text } = await get_kyou_content_text(asKyou(kyou), makeApi())
        expect(text).toBe('1行目\n\n2行目')
    })
})

describe('get_kyou_content_text 参照先をたどる種別', () => {
    test('ReKyou は参照先の本文を返す', async () => {
        const kyou = makeKyou({
            data_type: 'rekyou',
            typed_rekyou: makeReKyou({ attached_kyou: makeKyouWithKmemo('参照先のメモ') }),
        })
        const { text } = await get_kyou_content_text(asKyou(kyou), makeApi())
        expect(text).toBe('参照先のメモ')
    })

    test('MiReKyou は参照先の本文を返す', async () => {
        const kyou = makeKyou({
            data_type: 'mirekyou_create',
            typed_mirekyou: makeMiReKyou({ attached_kyou: makeKyouWithKmemo('タスク化した記録') }),
        })
        const { text } = await get_kyou_content_text(asKyou(kyou), makeApi())
        expect(text).toBe('タスク化した記録')
    })

    test('参照先が未取得なら取りに行って本文を返す', async () => {
        // attached_kyou は load_typed_datas では埋まらないので、無ければここで解決する
        const api = makeApi('', makeKyouWithKmemo('あとから取ってきたメモ'))
        const kyou = makeKyou({
            data_type: 'mirekyou_create',
            typed_mirekyou: makeMiReKyou({ attached_kyou: null, target_id: 'nested-target' }),
        })

        const { text } = await get_kyou_content_text(asKyou(kyou), api)

        expect(api.get_kyou).toHaveBeenCalledWith(expect.objectContaining({ id: 'nested-target' }))
        expect(text).toBe('あとから取ってきたメモ')
    })

    test('解決した参照先は attached_kyou に載せて2回目は引き直さない', async () => {
        const api = makeApi('', makeKyouWithKmemo('一度だけ取る'))
        const mirekyou = makeMiReKyou({ attached_kyou: null })
        const kyou = makeKyou({ data_type: 'mirekyou_create', typed_mirekyou: mirekyou })

        await get_kyou_content_text(asKyou(kyou), api)
        await get_kyou_content_text(asKyou(kyou), api)

        expect(api.get_kyou).toHaveBeenCalledTimes(1)
        expect(mirekyou.attached_kyou).not.toBeNull()
    })

    test('参照先が見つからなければ空文字。エラーにはしない', async () => {
        const api = makeApi()
        const kyou = makeKyou({
            data_type: 'mirekyou_create',
            typed_mirekyou: makeMiReKyou({ attached_kyou: null }),
        })

        const { text, errors } = await get_kyou_content_text(asKyou(kyou), api)

        expect(text).toBe('')
        expect(errors).toEqual([])
    })

    test('get_kyou が失敗しても空文字。エラーにはしない', async () => {
        const api = makeApi()
        api.get_kyou.mockResolvedValue({ errors: [{ error_message: 'ng' }], kyou_histories: [] })
        const kyou = makeKyou({
            data_type: 'mirekyou_create',
            typed_mirekyou: makeMiReKyou({ attached_kyou: null }),
        })

        const { text, errors } = await get_kyou_content_text(asKyou(kyou), api)

        expect(text).toBe('')
        expect(errors).toEqual([])
    })

    test('max_lazy_depth:1 なら1段だけ取りに行く', async () => {
        // 一覧の行から呼ぶときの設定。直列リクエストが深さぶん伸びないことを担保する
        const api = makeApi('', makeKyou({
            data_type: 'rekyou',
            typed_rekyou: makeReKyou({ attached_kyou: null, target_id: 'grandchild' }),
        }))
        const kyou = makeKyou({
            data_type: 'mirekyou_create',
            typed_mirekyou: makeMiReKyou({ attached_kyou: null }),
        })

        const { text } = await get_kyou_content_text(asKyou(kyou), api, 0, { max_lazy_depth: 1 })

        expect(api.get_kyou).toHaveBeenCalledTimes(1)
        expect(text).toBe('')
    })

    test('MiReKyou が自分を参照していても深さ上限で止まる', async () => {
        const kyou: Record<string, unknown> = makeKyou({ data_type: 'mirekyou_create' })
        kyou.typed_mirekyou = makeMiReKyou({ attached_kyou: kyou })

        const { text } = await get_kyou_content_text(asKyou(kyou), makeApi())
        expect(text).toBe('')
    })

    test('MiReKyou が ReKyou を参照していてもたどれる', async () => {
        const kyou = makeKyou({
            data_type: 'mirekyou_create',
            typed_mirekyou: makeMiReKyou({
                attached_kyou: makeKyou({
                    data_type: 'rekyou',
                    typed_rekyou: makeReKyou({ attached_kyou: makeKyouWithKmemo('孫のメモ') }),
                }),
            }),
        })
        const { text } = await get_kyou_content_text(asKyou(kyou), makeApi())
        expect(text).toBe('孫のメモ')
    })
})

describe('get_kyou_content_text プラグイン', () => {
    const plugin_kyou = () => makeKyou({ data_type: 'my_plugin', typed_plugin: { rep_name: 'my_plugin_rep' } })

    test('既定ではContent HTMLを取りに行き、テキストを抜き出す', async () => {
        const api = makeApi('<style>p{color:red}</style><p>プラグインの本文</p><script>alert(1)</script>')
        const { text } = await get_kyou_content_text(asKyou(plugin_kyou()), api)

        expect(api.get_plugin_content_html).toHaveBeenCalledTimes(1)
        expect(text).toBe('プラグインの本文')
    })

    test('allow_remote:false ならHTMLを取りに行かずリポジトリ名を返す', async () => {
        // 一覧の行から呼ぶときに件数ぶんリクエストが走らないことを担保する
        const api = makeApi('<p>取りに行ってはいけない</p>')
        const { text } = await get_kyou_content_text(asKyou(plugin_kyou()), api, 0, { allow_remote: false })

        expect(api.get_plugin_content_html).not.toHaveBeenCalled()
        expect(text).toBe('my_plugin_rep')
    })

    test('allow_remote:false は参照先をたどった先のプラグインにも効く', async () => {
        const api = makeApi('<p>取りに行ってはいけない</p>')
        const kyou = makeKyou({
            data_type: 'mirekyou_create',
            typed_mirekyou: makeMiReKyou({ attached_kyou: plugin_kyou() }),
        })
        const { text } = await get_kyou_content_text(asKyou(kyou), api, 0, { allow_remote: false })

        expect(api.get_plugin_content_html).not.toHaveBeenCalled()
        expect(text).toBe('my_plugin_rep')
    })

    test('Content HTMLの取得に失敗したらエラーを返す', async () => {
        const api = makeApi()
        api.get_plugin_content_html.mockResolvedValue({ errors: [{ error_message: 'ng' }], html: '' })

        const { text, errors } = await get_kyou_content_text(asKyou(plugin_kyou()), api)
        expect(text).toBe('')
        expect(errors).toHaveLength(1)
    })
})
