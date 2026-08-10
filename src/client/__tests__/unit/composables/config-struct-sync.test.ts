/**
 * 板ツリー・タグツリーのセッション中追随（useConfigStructSync）の検証。
 *
 * 板もタグもサーバに実体が無く、一覧APIから ApplicationConfig が起動時に
 * ツリーへ流し込んでいる導出概念なので、追随を落とすと
 * 「タスクを登録したのに板名ドロップダウンに出てこない」形の不具合になる。
 * 画面はエラーを出さず、再読込すると直ってしまうため再現も難しい。
 *
 * ここで固定するのは次の4点:
 *   - 板を持たない種別(kmemo等)には get_kyou を投げない（無駄な往復の抑止）
 *   - "mirekyou" は "mi" に前方一致するので MiReKyou を先に判定する
 *   - 既存の板/タグならツリーの取り直しも保存もしない
 *   - 増えていないときは clone() しない（identity を見る watch を無駄に走らせない）
 */
import { describe, expect, test, vi, type Mock } from 'vitest'

vi.mock('@/i18n', () => ({
    default: { global: { t: (key: string) => key, locale: 'ja' } },
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

// req_res は GkillAPIRequest を継承し、その constructor が GkillAPI.get_instance() を呼ぶ。
// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importを避けるため差し替える
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

import { ref, type Ref } from 'vue'
import { GkillError } from '@/classes/api/gkill-error'
import { useConfigStructSync } from '@/classes/use-config-struct-sync'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { GkillAPI } from '@/classes/api/gkill-api'
import type { GetAllTagNamesRequest } from '@/classes/api/req_res/get-all-tag-names-request'
import type { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import type { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'

// ── 小道具 ──

// タグツリー/板ツリーの両方に使えるノード。存在判定は tag_name / board_name を見る
interface FakeStructNode {
    key: string
    name: string
    tag_name: string
    board_name: string
    children: Array<FakeStructNode> | null
}

function struct_node(name: string, children: Array<FakeStructNode> | null = null): FakeStructNode {
    return {
        key: name,
        name: name,
        tag_name: name,
        board_name: name,
        children: children,
    }
}

// ApplicationConfig の実物は req_res との循環importを引き込むため、
// コンポーザブルが触るフィールドだけ持つ fake を使う。
// clone は spy にしてあり「identity を差し替えたか」を検査できる
interface FakeApplicationConfig {
    serial: number
    tag_struct: FakeStructNode
    mi_board_struct: FakeStructNode
    append_not_found_tags: Mock
    append_not_found_mi_boards: Mock
    clone: Mock
}

interface FakeKyou {
    id: string
    data_type: string
    typed_mi: { board_name: string } | null
    typed_mirekyou: { board_name: string } | null
    load_typed_mi: Mock
    load_typed_mirekyou: Mock
}

function make_fake_kyou(options: {
    data_type: string,
    mi_board_name?: string,
    mirekyou_board_name?: string,
}): FakeKyou {
    const kyou: FakeKyou = {
        id: 'kyou-id',
        data_type: options.data_type,
        typed_mi: null,
        typed_mirekyou: null,
        load_typed_mi: vi.fn(async () => {
            kyou.typed_mi = options.mi_board_name === undefined ? null : { board_name: options.mi_board_name }
            return []
        }),
        load_typed_mirekyou: vi.fn(async () => {
            kyou.typed_mirekyou = options.mirekyou_board_name === undefined ? null : { board_name: options.mirekyou_board_name }
            return []
        }),
    }
    return kyou
}

function make_error(error_code: string): GkillError {
    const error = new GkillError()
    error.error_code = error_code
    error.error_message = error_code
    return error
}

// 即時解決のPromiseチェーンを進める（in-flight 相乗りの検証で使う）
async function flush_microtasks(times = 30): Promise<void> {
    for (let i = 0; i < times; i++) {
        await Promise.resolve()
    }
}

function create_harness(setup?: { board_names?: Array<string>, tag_names?: Array<string> }) {
    const mi_board_struct = struct_node('__root__', (setup?.board_names ?? []).map(name => struct_node(name)))
    const tag_struct = struct_node('__root__', (setup?.tag_names ?? []).map(name => struct_node(name)))

    const append_not_found_tags = vi.fn(async (): Promise<Array<GkillError>> => [])
    const append_not_found_mi_boards = vi.fn(async (): Promise<Array<GkillError>> => [])

    let clone_serial = 0
    const clone = vi.fn((): FakeApplicationConfig => {
        clone_serial += 1
        return { ...config, serial: clone_serial }
    })

    const config: FakeApplicationConfig = {
        serial: 0,
        tag_struct: tag_struct,
        mi_board_struct: mi_board_struct,
        append_not_found_tags: append_not_found_tags,
        append_not_found_mi_boards: append_not_found_mi_boards,
        clone: clone,
    }

    let kyou_histories: Array<unknown> = []
    const get_kyou = vi.fn(async (_req: GetKyouRequest) => ({ kyou_histories: kyou_histories, messages: [], errors: [] }))
    const get_mi_board_list = vi.fn(async (_req: GetMiBoardRequest) => ({ boards: [], messages: [], errors: [] }))
    const get_all_tag_names = vi.fn(async (_req: GetAllTagNamesRequest) => ({ tag_names: [], messages: [], errors: [] }))
    const set_saved_application_config = vi.fn()

    const application_config = ref(config) as unknown as Ref<ApplicationConfig>
    const write_errors = vi.fn()

    const gkill_api = {
        get_kyou: get_kyou,
        get_mi_board_list: get_mi_board_list,
        get_all_tag_names: get_all_tag_names,
        set_saved_application_config: set_saved_application_config,
    }

    const sync = useConfigStructSync({
        application_config: application_config,
        gkill_api: () => gkill_api as unknown as GkillAPI,
        write_errors: write_errors,
    })

    return {
        sync,
        tag_struct,
        mi_board_struct,
        append_not_found_tags,
        append_not_found_mi_boards,
        clone,
        get_kyou,
        get_mi_board_list,
        get_all_tag_names,
        set_saved_application_config,
        write_errors,
        set_latest_kyou: (kyou: FakeKyou | null): void => {
            kyou_histories = kyou === null ? [] : [kyou]
        },
        current_serial: (): number => (application_config.value as unknown as FakeApplicationConfig).serial,
    }
}

function as_kyou(kyou: FakeKyou): Kyou {
    return kyou as unknown as Kyou
}

function as_tag(tag_name: string): Tag {
    return { tag: tag_name } as unknown as Tag
}

// ── check_mi_board_update ──

describe('check_mi_board_update: 板を持つ種別の絞り込み', () => {
    test('板を持たない種別(kmemo)には get_kyou を投げない', async () => {
        const harness = create_harness({ board_names: ['既存板'] })
        harness.set_latest_kyou(make_fake_kyou({ data_type: 'mi', mi_board_name: '新板' }))

        await harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: 'kmemo' })))

        expect(harness.get_kyou, '板を持たない種別で無駄な往復が出ている').not.toHaveBeenCalled()
        expect(harness.get_mi_board_list).not.toHaveBeenCalled()
    })

    test('data_type が空なら判断できないので通す', async () => {
        const harness = create_harness({ board_names: ['既存板'] })
        harness.set_latest_kyou(make_fake_kyou({ data_type: 'mi', mi_board_name: '既存板' }))

        await harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: '' })))

        expect(harness.get_kyou).toHaveBeenCalledTimes(1)
    })

    test('履歴が空なら板名の取得へ進まない', async () => {
        const harness = create_harness({ board_names: ['既存板'] })
        harness.set_latest_kyou(null)

        await harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: 'mi' })))

        expect(harness.get_kyou).toHaveBeenCalledTimes(1)
        expect(harness.get_mi_board_list).not.toHaveBeenCalled()
    })

    test('板名が空なら一覧の取り直しへ進まない', async () => {
        const harness = create_harness({ board_names: ['既存板'] })

        // typed_mi が読めなかった場合
        harness.set_latest_kyou(make_fake_kyou({ data_type: 'mi' }))
        await harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: 'mi' })))
        expect(harness.get_mi_board_list).not.toHaveBeenCalled()

        // 板名が空文字（既定の板へのフォールバック）の場合
        harness.set_latest_kyou(make_fake_kyou({ data_type: 'mi', mi_board_name: '' }))
        await harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: 'mi' })))
        expect(harness.get_mi_board_list).not.toHaveBeenCalled()
    })
})

describe('check_mi_board_update: MiReKyou と Mi の判定順', () => {
    test('mirekyou_* は typed_mirekyou の板名を使う（"mi" 前方一致より先に判定する）', async () => {
        // 「typed_mi 側の板名は既存 / typed_mirekyou 側の板名は未知」に仕込むことで、
        // 判定順を逆にすると「既存板なので何もしない」に化けて追随が止まるようにしてある
        const harness = create_harness({ board_names: ['板Mi'] })
        const latest = make_fake_kyou({
            data_type: 'mirekyou_create',
            mi_board_name: '板Mi',
            mirekyou_board_name: '板MiReKyou',
        })
        harness.set_latest_kyou(latest)

        await harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: 'mirekyou_create' })))

        expect(latest.load_typed_mirekyou, 'MiReKyou の板名を読んでいない').toHaveBeenCalledTimes(1)
        expect(latest.load_typed_mi, 'MiReKyou なのに Mi として読んでいる').not.toHaveBeenCalled()
        expect(harness.append_not_found_mi_boards, 'MiReKyou で入力した板名を取りこぼしている').toHaveBeenCalledTimes(1)
    })

    test('mi_* は typed_mi の板名を使う', async () => {
        const harness = create_harness({ board_names: ['板MiReKyou'] })
        const latest = make_fake_kyou({
            data_type: 'mi_create',
            mi_board_name: '板Mi',
            mirekyou_board_name: '板MiReKyou',
        })
        harness.set_latest_kyou(latest)

        await harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: 'mi_create' })))

        expect(latest.load_typed_mi).toHaveBeenCalledTimes(1)
        expect(latest.load_typed_mirekyou).not.toHaveBeenCalled()
        expect(harness.append_not_found_mi_boards).toHaveBeenCalledTimes(1)
    })
})

describe('check_mi_board_update: ツリーの取り直し条件', () => {
    test('既に板ツリーにある板ならツリーの取り直しも保存もしない', async () => {
        const harness = create_harness({ board_names: ['既存板'] })
        harness.set_latest_kyou(make_fake_kyou({ data_type: 'mi', mi_board_name: '既存板' }))

        await harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: 'mi' })))

        expect(harness.get_mi_board_list.mock.calls[0][0].force_reget, '一覧キャッシュの取り直しは常に行う').toBe(true)
        expect(harness.append_not_found_mi_boards).not.toHaveBeenCalled()
        expect(harness.clone).not.toHaveBeenCalled()
        expect(harness.set_saved_application_config).not.toHaveBeenCalled()
    })

    test('未知の板ならツリーへ足し、identity を差し替えて保存する', async () => {
        const harness = create_harness({ board_names: ['既存板'] })
        harness.append_not_found_mi_boards.mockImplementation(async () => {
            harness.mi_board_struct.children?.push(struct_node('新板'))
            return []
        })
        harness.set_latest_kyou(make_fake_kyou({ data_type: 'mi', mi_board_name: '新板' }))

        await harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: 'mi' })))

        expect(harness.get_mi_board_list.mock.calls[0][0].force_reget).toBe(true)
        expect(harness.append_not_found_mi_boards).toHaveBeenCalledTimes(1)
        expect(harness.clone, 'clone しないと identity を見る watch が走らず、板名ドロップダウンが古いままになる').toHaveBeenCalledTimes(1)
        expect(harness.current_serial()).toBe(1)
        expect(harness.set_saved_application_config).toHaveBeenCalledTimes(1)
    })

    test('取り直しても板が増えなければ clone しない', async () => {
        // 一覧に載っていない板（自分だけが知っている板）を渡したケース。
        // 無条件に clone すると識別子が変わり、全画面のフォームが引き直しになる
        const harness = create_harness({ board_names: ['既存板'] })
        harness.set_latest_kyou(make_fake_kyou({ data_type: 'mi', mi_board_name: '一覧に出ない板' }))

        await harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: 'mi' })))

        expect(harness.append_not_found_mi_boards).toHaveBeenCalledTimes(1)
        expect(harness.clone, '増えていないのに clone している').not.toHaveBeenCalled()
        expect(harness.set_saved_application_config).not.toHaveBeenCalled()
    })

    test('連打しても更新中の1回に相乗りする', async () => {
        const harness = create_harness({ board_names: ['既存板'] })
        let open_gate: () => void = () => { }
        const gate = new Promise<void>(resolve => { open_gate = resolve })
        harness.append_not_found_mi_boards.mockImplementation(async () => {
            await gate
            harness.mi_board_struct.children?.push(struct_node('新板'))
            return []
        })
        harness.set_latest_kyou(make_fake_kyou({ data_type: 'mi', mi_board_name: '新板' }))

        const first = harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: 'mi' })))
        const second = harness.sync.check_mi_board_update(as_kyou(make_fake_kyou({ data_type: 'mi' })))
        await flush_microtasks()
        open_gate()
        await Promise.all([first, second])

        expect(harness.append_not_found_mi_boards, '連打のぶんだけツリーを取り直している').toHaveBeenCalledTimes(1)
        expect(harness.clone, '相乗り側でも clone してしまい identity が2回変わっている').toHaveBeenCalledTimes(1)
    })
})

// ── check_tag_update ──

describe('check_tag_update', () => {
    test('タグ名が空なら何もしない', async () => {
        const harness = create_harness({ tag_names: ['既存タグ'] })

        await harness.sync.check_tag_update(as_tag(''))

        expect(harness.get_all_tag_names).not.toHaveBeenCalled()
        expect(harness.append_not_found_tags).not.toHaveBeenCalled()
    })

    test('既にタグツリーにあるタグならツリーの取り直しも保存もしない', async () => {
        const harness = create_harness({ tag_names: ['既存タグ'] })

        await harness.sync.check_tag_update(as_tag('既存タグ'))

        expect(harness.get_all_tag_names.mock.calls[0][0].force_reget).toBe(true)
        expect(harness.append_not_found_tags).not.toHaveBeenCalled()
        expect(harness.clone).not.toHaveBeenCalled()
        expect(harness.set_saved_application_config).not.toHaveBeenCalled()
    })

    test('未知のタグならツリーへ足し、identity を差し替えて保存する', async () => {
        const harness = create_harness({ tag_names: ['既存タグ'] })
        harness.append_not_found_tags.mockImplementation(async () => {
            harness.tag_struct.children?.push(struct_node('新タグ'))
            return []
        })

        await harness.sync.check_tag_update(as_tag('新タグ'))

        expect(harness.append_not_found_tags).toHaveBeenCalledTimes(1)
        expect(harness.clone).toHaveBeenCalledTimes(1)
        expect(harness.current_serial()).toBe(1)
        expect(harness.set_saved_application_config).toHaveBeenCalledTimes(1)
    })

    test('取り直しがエラーなら write_errors して clone しない', async () => {
        const harness = create_harness({ tag_names: ['既存タグ'] })
        const errors = [make_error('ERR000001')]
        harness.append_not_found_tags.mockImplementation(async () => {
            // 失敗しているのにツリーだけ増えている最悪ケースでも clone してはいけない
            harness.tag_struct.children?.push(struct_node('新タグ'))
            return errors
        })

        await harness.sync.check_tag_update(as_tag('新タグ'))

        expect(harness.write_errors).toHaveBeenCalledWith(errors)
        expect(harness.clone, 'エラーなのに identity を差し替えている').not.toHaveBeenCalled()
        expect(harness.set_saved_application_config).not.toHaveBeenCalled()
    })
})

// ── resync_structs ──

describe('resync_structs', () => {
    test('タグ・板の両方を force_reget で取り直し、両ツリーを更新する', async () => {
        // KFTL は保存したタグを registered_tag で上げてこないので、
        // 片方だけ取り直すと KFTL 経由で作ったタグ or 板がその画面から消える
        const harness = create_harness()

        await harness.sync.resync_structs()

        expect(harness.get_all_tag_names).toHaveBeenCalledTimes(1)
        expect(harness.get_all_tag_names.mock.calls[0][0].force_reget).toBe(true)
        expect(harness.get_mi_board_list).toHaveBeenCalledTimes(1)
        expect(harness.get_mi_board_list.mock.calls[0][0].force_reget).toBe(true)
        expect(harness.append_not_found_tags, 'タグツリーを取り直していない').toHaveBeenCalledTimes(1)
        expect(harness.append_not_found_mi_boards, '板ツリーを取り直していない').toHaveBeenCalledTimes(1)
        expect(harness.get_kyou, 'resync では個別の Kyou を引かない').not.toHaveBeenCalled()
    })

    test('両ツリーが増えたときは clone が1回ずつ走る', async () => {
        const harness = create_harness()
        harness.append_not_found_tags.mockImplementation(async () => {
            harness.tag_struct.children?.push(struct_node('新タグ'))
            return []
        })
        harness.append_not_found_mi_boards.mockImplementation(async () => {
            harness.mi_board_struct.children?.push(struct_node('新板'))
            return []
        })

        await harness.sync.resync_structs()

        expect(harness.clone).toHaveBeenCalledTimes(2)
        expect(harness.current_serial()).toBe(2)
        expect(harness.set_saved_application_config).toHaveBeenCalledTimes(2)
    })
})
