/**
 * Kyou 引き直しの共通手順のテスト。
 *
 * 以前は6箇所に手書きでコピーされていて、手順が完全に一致していたのは rykv の列ループだけだった。
 * とくに `load_all` の第2引数(force_attached)を落としている実装では、
 * `Kyou.clone()` が `is_attached_tags_loaded` を引き継ぐせいで
 * `InfoBase.load_attached_tags(false)` が早期returnし、添付タグを一度も引き直さない。
 * 「タグを足しても表示が変わらない」というバグの正体がこれなので、
 * 手順と引数をここで固定する。
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/classes/delete-gkill-cache', () => ({
    default: vi.fn(),
}))

import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { build_mi_reload_query, is_kyou_reloading, new_reload_batch, refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'

const cache_delete_mock = delete_gkill_kyou_cache as unknown as ReturnType<typeof vi.fn>

interface FakeKyouOptions {
    reload_throws?: boolean
    /** 引き直しを飛行中のまま止めておくための関門。resolve するまで reload が返らない */
    reload_gate?: Promise<void>
}

/**
 * 本物の Kyou の「clone が is_typed_data_loaded を引き継ぐ」性質を再現した偽物。
 * 呼び出し順は共有の calls 配列に積むので、clone をまたいで観測できる。
 *
 * abort_controller は本物の clone() と違って写す。引き直しが呼び出し元の
 * controller を引き継いでいたら reload が失敗するようにしてあるので、
 * kyou-reload 側で切り離すのをやめたらテストが落ちる。
 */
function make_kyou(id: string, calls: Array<string>, options?: FakeKyouOptions) {
    const kyou = {
        id: id,
        is_typed_data_loaded: true,
        is_attached_tags_loaded: true,
        abort_controller: new AbortController(),
        async reload(is_updated_info: boolean, _query?: FindKyouQuery): Promise<Array<never>> {
            calls.push(`reload:${is_updated_info}`)
            if (options?.reload_gate) {
                await options.reload_gate
            }
            if (kyou.abort_controller.signal.aborted) {
                throw new Error('aborted')
            }
            if (options?.reload_throws) {
                throw new Error('reload failed')
            }
            return []
        },
        async load_all(_query?: FindKyouQuery, force_attached = false): Promise<Array<never>> {
            // force_attached と、その時点の is_typed_data_loaded を一緒に記録する。
            // 「フラグを倒してから load_all する」順序が崩れたらここで落ちる
            calls.push(`load_all:${force_attached}:typed_loaded=${kyou.is_typed_data_loaded}`)
            return []
        },
        clone() {
            const cloned = make_kyou(id, calls, options)
            cloned.is_typed_data_loaded = kyou.is_typed_data_loaded
            cloned.is_attached_tags_loaded = kyou.is_attached_tags_loaded
            cloned.abort_controller = kyou.abort_controller
            return cloned
        },
    }
    return kyou
}

function as_kyou(fake: ReturnType<typeof make_kyou>): Kyou {
    return fake as unknown as Kyou
}

/** resolve を外から呼べる関門 */
function make_gate(): { gate: Promise<void>, open: () => void } {
    let open = (): void => { }
    const gate = new Promise<void>((resolve) => { open = () => resolve() })
    return { gate: gate, open: open }
}

/** performance.now() が確実に進むまで待つ */
function tick(): Promise<void> {
    return new Promise<void>((resolve) => { setTimeout(resolve, 2) })
}

beforeEach(() => {
    cache_delete_mock.mockClear()
    cache_delete_mock.mockImplementation(() => Promise.resolve())
})

describe('refresh_kyou', () => {
    it('キャッシュ削除 → reload(true) → フラグを倒す → load_all(force_attached=true) の順で引き直す', async () => {
        const calls = new Array<string>()
        const kyou = make_kyou('kyou-1', calls)

        const refreshed = await refresh_kyou(as_kyou(kyou))

        expect(cache_delete_mock).toHaveBeenCalledWith('kyou-1')
        expect(calls).toEqual([
            'reload:true',
            'load_all:true:typed_loaded=false',
        ])
        expect(refreshed).not.toBeNull()
    })

    it('引数のKyouは変更しない', async () => {
        const calls = new Array<string>()
        const kyou = make_kyou('kyou-1', calls)

        await refresh_kyou(as_kyou(kyou))

        expect(kyou.is_typed_data_loaded).toBe(true)
    })

    it('Cache APIが使えなくても引き直しは続ける', async () => {
        cache_delete_mock.mockImplementation(() => Promise.reject(new Error('no caches')))
        const calls = new Array<string>()
        const kyou = make_kyou('kyou-1', calls)

        const refreshed = await refresh_kyou(as_kyou(kyou))

        expect(refreshed).not.toBeNull()
        expect(calls).toContain('reload:true')
    })

    it('reloadが失敗したら null を返す', async () => {
        const calls = new Array<string>()
        const kyou = make_kyou('kyou-1', calls, { reload_throws: true })

        const refreshed = await refresh_kyou(as_kyou(kyou))

        expect(refreshed).toBeNull()
    })

    // 1回のタグ追加でリスト・focused・開いているダイアログが独立に引き直すため、
    // 合流しないと同じKyouに対して何往復もリクエストが飛ぶ
    it('同じ更新から派生した呼び出し(requested_atが同じ)は1回にまとめる', async () => {
        const calls = new Array<string>()
        const requested_at = new_reload_batch()

        const [a, b] = await Promise.all([
            refresh_kyou(as_kyou(make_kyou('kyou-1', calls)), undefined, requested_at),
            refresh_kyou(as_kyou(make_kyou('kyou-1', calls)), undefined, requested_at),
        ])

        expect(calls.filter(call => call.startsWith('reload:'))).toHaveLength(1)
        expect(cache_delete_mock).toHaveBeenCalledTimes(1)
        // 同一インスタンスを配り回すと後段の読み込みで副作用が出るので、別インスタンスを返す
        expect(a).not.toBe(b)
    })

    // ダイアログを開いたときの引き直しがまだ飛行中のうちに保存すると、そこへ相乗りして
    // 更新前のKyouを配り、列・focused・ダイアログを一斉に古い内容へ戻していた
    it('自分より前に始まった引き直しには相乗りしない', async () => {
        const calls = new Array<string>()
        const { gate, open } = make_gate()

        // ダイアログを開いたときの引き直し(まだ飛行中)
        const in_flight = refresh_kyou(as_kyou(make_kyou('kyou-1', calls, { reload_gate: gate })))
        await tick()
        // そのあとの保存で要求された引き直し
        const after_write = refresh_kyou(as_kyou(make_kyou('kyou-1', calls)))

        open()
        await Promise.all([in_flight, after_write])

        expect(calls.filter(call => call.startsWith('reload:'))).toHaveLength(2)
    })

    // 保存してダイアログを閉じると、そのダイアログのKyouViewがunmountされてabortする。
    // 引き直しがそれに巻き込まれると「閉じたら更新されない」になる
    it('呼び出し元のKyouをabortしても引き直しは完走する', async () => {
        const calls = new Array<string>()
        const { gate, open } = make_gate()
        const kyou = make_kyou('kyou-1', calls, { reload_gate: gate })

        const promise = refresh_kyou(as_kyou(kyou))
        kyou.abort_controller.abort()
        open()

        expect(await promise).not.toBeNull()
    })

    it('決着後は合流しない（次の呼び出しは引き直す）', async () => {
        const calls = new Array<string>()
        await refresh_kyou(as_kyou(make_kyou('kyou-1', calls)))
        await refresh_kyou(as_kyou(make_kyou('kyou-1', calls)))

        expect(calls.filter(call => call.startsWith('reload:'))).toHaveLength(2)
    })

    it('idが違えば合流しない', async () => {
        const calls = new Array<string>()

        await Promise.all([
            refresh_kyou(as_kyou(make_kyou('kyou-1', calls))),
            refresh_kyou(as_kyou(make_kyou('kyou-2', calls))),
        ])

        expect(calls.filter(call => call.startsWith('reload:'))).toHaveLength(2)
    })
})

// Kyou単位のぐるぐるはこのフラグで出す。KyouViewはidが同じなら再マウントされないので、
// 状態はコンポーネントではなくid単位で持つ必要がある
describe('is_kyou_reloading', () => {
    it('引き直しの間だけ true になる', async () => {
        const calls = new Array<string>()
        const { gate, open } = make_gate()
        const kyou = make_kyou('kyou-1', calls, { reload_gate: gate })

        expect(is_kyou_reloading('kyou-1')).toBe(false)

        const promise = refresh_kyou(as_kyou(kyou))
        expect(is_kyou_reloading('kyou-1')).toBe(true)

        open()
        await promise
        expect(is_kyou_reloading('kyou-1')).toBe(false)
    })

    it('相乗りした側も数えるので、1本終わっただけでは倒れない', async () => {
        const calls = new Array<string>()
        const { gate, open } = make_gate()
        const requested_at = new_reload_batch()

        const first = refresh_kyou(as_kyou(make_kyou('kyou-1', calls, { reload_gate: gate })), undefined, requested_at)
        const second = refresh_kyou(as_kyou(make_kyou('kyou-1', calls, { reload_gate: gate })), undefined, requested_at)
        expect(is_kyou_reloading('kyou-1')).toBe(true)

        open()
        await Promise.all([first, second])
        expect(is_kyou_reloading('kyou-1')).toBe(false)
    })

    it('引き直していないidでは false', () => {
        expect(is_kyou_reloading('kyou-not-reloading')).toBe(false)
    })
})

describe('refresh_kyou_in_list', () => {
    it('既定では配列をin-placeで差し替える（親の配列と縁を切らない）', async () => {
        const calls = new Array<string>()
        const target = make_kyou('kyou-1', calls)
        const other = make_kyou('kyou-2', calls)
        const list = [as_kyou(other), as_kyou(target)]
        const original_list = list

        await refresh_kyou_in_list(list, as_kyou(target))

        expect(list).toBe(original_list)
        expect(list[1]).not.toBe(as_kyou(target))
        expect(list[1].id).toBe('kyou-1')
        expect(list[0]).toBe(as_kyou(other))
    })

    it('replaceを渡したときは新しい配列を作って渡す（元の配列は触らない）', async () => {
        const calls = new Array<string>()
        const target = make_kyou('kyou-1', calls)
        const list = [as_kyou(target)]
        let replaced: Array<Kyou> | null = null

        await refresh_kyou_in_list(list, as_kyou(target), {
            replace: (next_list) => { replaced = next_list },
        })

        expect(replaced).not.toBeNull()
        expect(replaced).not.toBe(list)
        expect(list[0]).toBe(as_kyou(target))
    })

    it('idが一致する要素が無ければリクエストを飛ばさない', async () => {
        const calls = new Array<string>()
        const target = make_kyou('kyou-1', calls)
        const list = [as_kyou(make_kyou('kyou-2', calls))]

        await refresh_kyou_in_list(list, as_kyou(target))

        expect(calls).toHaveLength(0)
        expect(cache_delete_mock).not.toHaveBeenCalled()
    })

    it('引き直しに失敗したらリストを変更しない', async () => {
        const calls = new Array<string>()
        const target = make_kyou('kyou-1', calls, { reload_throws: true })
        const list = [as_kyou(target)]

        await refresh_kyou_in_list(list, as_kyou(target))

        expect(list[0]).toBe(as_kyou(target))
    })

    it('クエリを関数で渡すとリスト内の要素から導出できる', async () => {
        const calls = new Array<string>()
        const target = make_kyou('kyou-1', calls)
        const list = [as_kyou(target)]
        let resolved_with: Kyou | null = null

        await refresh_kyou_in_list(list, as_kyou(target), {
            query: (kyou_in_list) => { resolved_with = kyou_in_list; return undefined },
        })

        expect(resolved_with).toBe(as_kyou(target))
    })
})

describe('build_mi_reload_query', () => {
    function make_query(): FindKyouQuery {
        const query = {
            for_mi: false,
            mi_sort_type: MiSortType.estimate_start_time,
            clone(): FindKyouQuery {
                return make_query()
            },
        }
        return query as unknown as FindKyouQuery
    }

    it('mi以外のdata_typeではクエリを作らない', () => {
        expect(build_mi_reload_query(make_query(), 'kmemo_create')).toBeUndefined()
    })

    it('mi_* から並び順を決める', () => {
        expect(build_mi_reload_query(make_query(), 'mi_start')?.mi_sort_type).toBe(MiSortType.estimate_start_time)
        expect(build_mi_reload_query(make_query(), 'mi_end')?.mi_sort_type).toBe(MiSortType.estimate_end_time)
        expect(build_mi_reload_query(make_query(), 'mi_limit')?.mi_sort_type).toBe(MiSortType.limit_time)
        expect(build_mi_reload_query(make_query(), 'mi_create')?.mi_sort_type).toBe(MiSortType.create_time)
    })

    // mirekyou を mi より先に判定しないと "rekyou_start" が接頭辞として残る
    it('mirekyou_* も mi_* と同じ並び順になる', () => {
        expect(build_mi_reload_query(make_query(), 'mirekyou_limit')?.mi_sort_type).toBe(MiSortType.limit_time)
        expect(build_mi_reload_query(make_query(), 'mirekyou_end')?.mi_sort_type).toBe(MiSortType.estimate_end_time)
    })

    it('for_mi を立てる', () => {
        expect(build_mi_reload_query(make_query(), 'mi_create')?.for_mi).toBe(true)
    })
})
