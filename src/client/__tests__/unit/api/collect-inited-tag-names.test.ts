/**
 * 「初期チェックあり」タグ名の収集の検証。
 *
 * この集合は2箇所で共用されている:
 *   - FindKyouQuery の既定クエリ生成（起動直後に何が検索されるか）
 *   - TimeIsタグツリーの null フォールバック（use-time-is-query.ts の timeis_tags_for_tree）
 * どちらも「集合が空になっても画面はエラーを出さず、結果だけ変わる」ため、
 * 取りこぼしても気付けない。ここで集合そのものを固定する。
 *
 * 後半では use-time-is-query.ts の null フォールバック経路
 * （timeis_tags=null＝グループ未使用のときだけこの集合へ落ちる）を検証する。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('@/i18n', () => ({
    default: { global: { t: (key: string) => key, locale: 'ja' } },
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

vi.mock('@/classes/api/gkill-api', () => ({
    GkillAPI: {
        get_instance: vi.fn(() => ({
            get_session_id: vi.fn(() => 'mock-session'),
            generate_uuid: vi.fn(() => 'mock-uuid'),
        })),
        get_gkill_api: vi.fn(() => ({
            get_session_id: vi.fn(() => 'mock-session'),
            generate_uuid: vi.fn(() => 'mock-uuid'),
        })),
    },
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
    default: vi.fn().mockResolvedValue(undefined),
    delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
}))

import { nextTick, reactive } from 'vue'
import { collect_inited_tag_names } from '@/classes/api/find_query/collect-inited-tag-names'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { useTimeIsQuery } from '@/classes/use-time-is-query'
import type { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'

function tag_node(tag_name: string, options?: Partial<TagStructElementData>): TagStructElementData {
    return {
        name: tag_name,
        id: null,
        tag_name: tag_name,
        check_when_inited: false,
        is_force_hide: false,
        children: null,
        key: tag_name,
        is_checked: false,
        indeterminate: false,
        is_dir: false,
        ...options,
    }
}

describe('collect_inited_tag_names', () => {
    test('check_when_inited が立った子だけを集める', () => {
        const struct = tag_node('__root__', {
            is_dir: true,
            children: [
                tag_node('タグA', { check_when_inited: true }),
                tag_node('タグB'),
                tag_node('タグC', { check_when_inited: true }),
            ],
        })
        expect(collect_inited_tag_names(struct)).toEqual(['タグA', 'タグC'])
    })

    test('深い階層の初期チェックタグも集める（再帰していないと取りこぼす）', () => {
        const struct = tag_node('__root__', {
            is_dir: true,
            children: [
                tag_node('タグA', { check_when_inited: true }),
                tag_node('フォルダ', {
                    is_dir: true,
                    children: [
                        tag_node('タグB'),
                        tag_node('入れ子フォルダ', {
                            is_dir: true,
                            children: [
                                tag_node('タグC', { check_when_inited: true }),
                            ],
                        }),
                    ],
                }),
            ],
        })
        expect(collect_inited_tag_names(struct)).toEqual(['タグA', 'タグC'])
    })

    test('フォルダ自身に初期チェックが立っていればフォルダ名も入る（配下は別途walkされる）', () => {
        const struct = tag_node('__root__', {
            is_dir: true,
            children: [
                tag_node('フォルダ', {
                    is_dir: true,
                    check_when_inited: true,
                    children: [
                        tag_node('タグA', { check_when_inited: true }),
                    ],
                }),
            ],
        })
        expect(collect_inited_tag_names(struct)).toEqual(['フォルダ', 'タグA'])
    })

    test('ルート自身の check_when_inited は集めない（走査は子から始まる）', () => {
        const struct = tag_node('__root__', {
            is_dir: true,
            check_when_inited: true,
            children: [
                tag_node('タグA', { check_when_inited: true }),
            ],
        })
        expect(collect_inited_tag_names(struct), 'ルート(仮想ノード)のキーが検索条件へ混入している').toEqual(['タグA'])
    })

    test('同じタグ名が複数ノードにあれば出た数だけ入る（重複除去はしない）', () => {
        const struct = tag_node('__root__', {
            is_dir: true,
            children: [
                tag_node('タグA', { check_when_inited: true }),
                tag_node('フォルダ', {
                    is_dir: true,
                    children: [
                        tag_node('タグA', { check_when_inited: true }),
                    ],
                }),
            ],
        })
        expect(collect_inited_tag_names(struct)).toEqual(['タグA', 'タグA'])
    })

    test('children が null / 空なら空配列', () => {
        expect(collect_inited_tag_names(tag_node('__root__'))).toEqual([])
        expect(collect_inited_tag_names(tag_node('__root__', { children: [] }))).toEqual([])
    })

    test('初期チェックが1つも無ければ空配列', () => {
        const struct = tag_node('__root__', {
            is_dir: true,
            children: [tag_node('タグA'), tag_node('タグB')],
        })
        expect(collect_inited_tag_names(struct)).toEqual([])
    })
})

// ── useTimeIsQuery の null フォールバック ──

// ApplicationConfig の実物は req_res との循環importを引き込むため、
// コンポーザブルが触るフィールドだけ持つ fake を使う
function make_fake_application_config(): Record<string, unknown> {
    const config: Record<string, unknown> = {
        tag_struct: tag_node('__root__', {
            is_dir: true,
            children: [
                tag_node('タグA', { check_when_inited: true }),
                tag_node('タグB'),
                tag_node('フォルダ', {
                    is_dir: true,
                    children: [
                        tag_node('タグC', { check_when_inited: true }),
                    ],
                }),
            ],
        }),
        google_map_api_key: '',
    }
    config.clone = () => {
        const cloned: Record<string, unknown> = { ...config }
        cloned.tag_struct = JSON.parse(JSON.stringify(config.tag_struct))
        return cloned
    }
    return config
}

// props watcher（async含む）を消化する
async function flush_watchers(times = 4): Promise<void> {
    for (let i = 0; i < times; i++) {
        await nextTick()
    }
}

// ツリーへ実際にチェックが入ったキーを拾う
function collect_checked_keys(struct: unknown): Array<string> {
    const checked = new Array<string>()
    const walk = (node: TagStructElementData): void => {
        if (node.is_checked) {
            checked.push(node.key)
        }
        node.children?.forEach(child => walk(child))
    }
    walk(struct as TagStructElementData)
    return checked
}

describe('useTimeIsQuery: timeis_tags=null のときの初期チェック集合フォールバック', () => {
    function create_view() {
        const props = reactive({
            application_config: make_fake_application_config(),
            find_kyou_query: new FindKyouQuery(),
            inited: true,
        })
        const view = useTimeIsQuery({ props: props as never, emits: vi.fn() as never })
        return { props, view }
    }

    test('timeis_tags=null（グループ未使用）なら初期チェックタグがツリーへ反映される', async () => {
        const { props, view } = create_view()

        const query = new FindKyouQuery()
        query.query_id = 'q1'
        query.timeis_tags = null
        props.find_kyou_query = query
        await flush_watchers()

        expect(view.get_use_timeis_tags(), 'timeis_tags=null はチェックオフ（グループ未使用）').toBe(false)
        expect(
            collect_checked_keys(view.cloned_application_config.value.tag_struct),
            'null フォールバックが効いていないとツリーが全部オフになる',
        ).toEqual(['タグA', 'タグC'])
    })

    test('timeis_tags が非nullならその集合だけが反映される（フォールバックしない）', async () => {
        const { props, view } = create_view()

        const query = new FindKyouQuery()
        query.query_id = 'q1'
        query.timeis_words = []
        query.timeis_tags = ['タグB']
        props.find_kyou_query = query
        await flush_watchers()

        expect(view.get_use_timeis_tags()).toBe(true)
        expect(
            collect_checked_keys(view.cloned_application_config.value.tag_struct),
            '明示指定があるのに初期チェック集合へ落ちている',
        ).toEqual(['タグB'])
    })

    test('timeis_tags=[]（有効・0個指定）は空集合であってフォールバックではない', async () => {
        const { props, view } = create_view()

        const query = new FindKyouQuery()
        query.query_id = 'q1'
        query.timeis_tags = []
        props.find_kyou_query = query
        await flush_watchers()

        expect(view.get_use_timeis_tags()).toBe(true)
        expect(
            collect_checked_keys(view.cloned_application_config.value.tag_struct),
            '[] を null と同じ扱いにすると「0件指定」が表現できなくなる',
        ).toEqual([])
    })
})
