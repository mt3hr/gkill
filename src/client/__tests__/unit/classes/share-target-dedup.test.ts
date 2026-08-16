/**
 * Web Share Target の二重保存を防ぐ台帳のテスト。
 * serviceWorker.ts は import しただけで副作用が走るので、判定は全部 share-target-dedup.ts 側にある。
 */
import {
    SHARE_DEDUP_MAX_ENTRIES,
    SHARE_DEDUP_WINDOW_MILLI_SECONDS,
    append_share_ledger,
    decide_share_save_target,
    find_duplicated_share_entry,
    force_save_shared_payload,
    is_same_shared_payload,
    prune_share_ledger,
    read_share_ledger,
    type ShareLedgerEntry,
    type SharedPayload,
} from '@/classes/share-target-dedup'

function payload(overrides: Partial<SharedPayload> = {}): SharedPayload {
    return { title: '', text: '', url: '', ...overrides }
}

function entry(id: string, saved_at: number, p: SharedPayload): ShareLedgerEntry {
    return { id, saved_at, payload: p }
}

const now = 1_700_000_000_000

// ========== decide_share_save_target ==========

describe('decide_share_save_target', () => {
    test('url があれば urlog。title はそのままタイトルになる', () => {
        expect(decide_share_save_target(payload({ url: 'https://example.com/a', title: 'タイトル' })))
            .toEqual({ kind: 'urlog', url: 'https://example.com/a', title: 'タイトル' })
    })

    test('url が無く title が URL なら urlog。タイトルは空になる', () => {
        expect(decide_share_save_target(payload({ title: 'https://example.com/b' })))
            .toEqual({ kind: 'urlog', url: 'https://example.com/b', title: '' })
    })

    test('text 全体が URL なら urlog。title をタイトルにする', () => {
        expect(decide_share_save_target(payload({ text: 'https://example.com/c', title: 'ページ名' })))
            .toEqual({ kind: 'urlog', url: 'https://example.com/c', title: 'ページ名' })
    })

    test('AndroidのGoogleアプリのように末尾にURLが付いた text は末尾のURLを拾う', () => {
        expect(decide_share_save_target(payload({ text: '記事の見出し\nhttps://example.com/d' })))
            .toEqual({ kind: 'urlog', url: 'https://example.com/d', title: '' })
    })

    test('URLを含まない text は kmemo', () => {
        expect(decide_share_save_target(payload({ text: 'ただのメモ' })))
            .toEqual({ kind: 'kmemo', content: 'ただのメモ' })
    })

    test('1文字の text も kmemo（分割数が1のとき先頭を見る旧実装と結果が変わらない）', () => {
        expect(decide_share_save_target(payload({ text: 'あ' })))
            .toEqual({ kind: 'kmemo', content: 'あ' })
    })

    test('http を含むがURLにならない text は kmemo', () => {
        expect(decide_share_save_target(payload({ text: 'httpの話をした' })))
            .toEqual({ kind: 'kmemo', content: 'httpの話をした' })
    })

    test('http以外のスキームは URL とみなさない', () => {
        expect(decide_share_save_target(payload({ url: 'ftp://example.com/e', text: 'メモ' })))
            .toEqual({ kind: 'kmemo', content: 'メモ' })
    })

    test('全部空なら保存対象なし', () => {
        expect(decide_share_save_target(payload())).toBeNull()
    })
})

// ========== is_same_shared_payload ==========

describe('is_same_shared_payload', () => {
    test('3フィールドが全部同じなら同じ', () => {
        expect(is_same_shared_payload(
            payload({ title: 't', text: 'x', url: 'https://example.com/' }),
            payload({ title: 't', text: 'x', url: 'https://example.com/' }),
        )).toBe(true)
    })

    test('title だけ違えば別', () => {
        expect(is_same_shared_payload(payload({ title: 'a' }), payload({ title: 'b' }))).toBe(false)
    })

    test('text だけ違えば別', () => {
        expect(is_same_shared_payload(payload({ text: 'a' }), payload({ text: 'b' }))).toBe(false)
    })

    test('url だけ違えば別', () => {
        expect(is_same_shared_payload(
            payload({ url: 'https://example.com/a' }),
            payload({ url: 'https://example.com/b' }),
        )).toBe(false)
    })
})

// ========== find_duplicated_share_entry ==========

describe('find_duplicated_share_entry', () => {
    const shared = payload({ url: 'https://example.com/dup' })

    test('期間内に同じ内容があれば見つかる', () => {
        const entries = [entry('id1', now - 1000, shared)]
        expect(find_duplicated_share_entry(entries, shared, now)?.id).toBe('id1')
    })

    test('期間を過ぎていたら見つからない（意図的な再共有として保存させる）', () => {
        const entries = [entry('id1', now - SHARE_DEDUP_WINDOW_MILLI_SECONDS, shared)]
        expect(find_duplicated_share_entry(entries, shared, now)).toBeNull()
    })

    test('内容が違えば見つからない', () => {
        const entries = [entry('id1', now - 1000, payload({ url: 'https://example.com/other' }))]
        expect(find_duplicated_share_entry(entries, shared, now)).toBeNull()
    })

    test('空の台帳では見つからない', () => {
        expect(find_duplicated_share_entry([], shared, now)).toBeNull()
    })
})

// ========== prune_share_ledger ==========

describe('prune_share_ledger', () => {
    test('期限切れを落とす', () => {
        const entries = [
            entry('new', now - 1000, payload({ url: 'https://example.com/1' })),
            entry('old', now - SHARE_DEDUP_WINDOW_MILLI_SECONDS - 1, payload({ url: 'https://example.com/2' })),
        ]
        expect(prune_share_ledger(entries, now).map((e) => e.id)).toEqual(['new'])
    })

    test('新しい順に並べる', () => {
        const entries = [
            entry('a', now - 3000, payload({ url: 'https://example.com/a' })),
            entry('c', now - 1000, payload({ url: 'https://example.com/c' })),
            entry('b', now - 2000, payload({ url: 'https://example.com/b' })),
        ]
        expect(prune_share_ledger(entries, now).map((e) => e.id)).toEqual(['c', 'b', 'a'])
    })

    test('上限件数で切る', () => {
        const entries = Array.from({ length: SHARE_DEDUP_MAX_ENTRIES + 10 }, (_, i) =>
            entry(`id${i}`, now - i * 1000, payload({ url: `https://example.com/${i}` })))
        const pruned = prune_share_ledger(entries, now)
        expect(pruned).toHaveLength(SHARE_DEDUP_MAX_ENTRIES)
        expect(pruned[0].id).toBe('id0')
    })

    test('元の配列を書き換えない', () => {
        const entries = [
            entry('a', now - 3000, payload({ url: 'https://example.com/a' })),
            entry('b', now - 1000, payload({ url: 'https://example.com/b' })),
        ]
        prune_share_ledger(entries, now)
        expect(entries.map((e) => e.id)).toEqual(['a', 'b'])
    })
})

// ========== append_share_ledger ==========

describe('append_share_ledger', () => {
    test('先頭に積む', () => {
        const entries = [entry('old', now - 1000, payload({ url: 'https://example.com/old' }))]
        const added = entry('new', now, payload({ url: 'https://example.com/new' }))
        expect(append_share_ledger(entries, added, now).map((e) => e.id)).toEqual(['new', 'old'])
    })

    test('同じ id の古い行は差し替える（強制保存で saved_at を進める経路）', () => {
        const shared = payload({ url: 'https://example.com/dup' })
        const entries = [entry('id1', now - 10_000, shared)]
        const result = append_share_ledger(entries, entry('id1', now, shared), now)
        expect(result).toHaveLength(1)
        expect(result[0].saved_at).toBe(now)
    })

    test('積むついでに期限切れも落とす', () => {
        const entries = [entry('old', now - SHARE_DEDUP_WINDOW_MILLI_SECONDS - 1, payload({ url: 'https://example.com/old' }))]
        const added = entry('new', now, payload({ url: 'https://example.com/new' }))
        expect(append_share_ledger(entries, added, now).map((e) => e.id)).toEqual(['new'])
    })
})

// ========== Cache Storage が無い環境 ==========

describe('read_share_ledger', () => {
    test('Cache Storage が無ければ空で返す（落とさない）', async () => {
        expect(await read_share_ledger()).toEqual([])
    })
})

// ========== force_save_shared_payload ==========

describe('force_save_shared_payload', () => {
    afterEach(() => {
        vi.unstubAllGlobals()
    })

    test('is_saved:true が返れば true', async () => {
        vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ is_saved: true }), {
            status: 200, headers: { 'Content-Type': 'application/json' },
        })))
        expect(await force_save_shared_payload(payload({ url: 'https://example.com/x' }))).toBe(true)
    })

    test('is_saved:false が返れば false', async () => {
        vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ is_saved: false }), {
            status: 200, headers: { 'Content-Type': 'application/json' },
        })))
        expect(await force_save_shared_payload(payload({ url: 'https://example.com/x' }))).toBe(false)
    })

    test('ServiceWorker が居ないと HTML や 404 が返るので false', async () => {
        vi.stubGlobal('fetch', vi.fn(async () => new Response('<!doctype html>', { status: 200 })))
        expect(await force_save_shared_payload(payload({ url: 'https://example.com/x' }))).toBe(false)
    })

    test('通信そのものが失敗しても投げずに false', async () => {
        vi.stubGlobal('fetch', vi.fn(async () => { throw new Error('offline') }))
        expect(await force_save_shared_payload(payload({ url: 'https://example.com/x' }))).toBe(false)
    })
})
