/**
 * メモ帳のタブを持つ共有ストアのテスト。
 *
 * ストアはモジュールシングルトンで、`/mkfl` のように KFTLView が2つ同時に
 * マウントされる画面でも真実を1つに保つためのもの。
 * 生存期間がコンポーネントより長いことと、タブが常に1枚以上あることを固定する。
 */
import { afterEach, beforeEach, describe, test, expect, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick } from 'vue'

import { reset_kftl_tabs_for_test, useKftlTabs } from '@/classes/use-kftl-tabs'
import { KFTL_TABS_STORAGE_KEY, parse_kftl_tabs } from '@/classes/kftl-tabs'

beforeEach(() => {
    localStorage.clear()
    reset_kftl_tabs_for_test()
})

afterEach(() => {
    reset_kftl_tabs_for_test()
    vi.restoreAllMocks()
})

/** setup の中でストアを取り、その app を返す（unmount できるように） */
function mount_store_host() {
    let store: ReturnType<typeof useKftlTabs> | null = null
    const Host = defineComponent({
        setup() {
            store = useKftlTabs()
            return () => h('div')
        },
    })
    const app = createApp(Host)
    app.mount(document.createElement('div'))
    return { app, store: store! }
}

function saved_tabs() {
    const raw = localStorage.getItem(KFTL_TABS_STORAGE_KEY)
    expect(raw, 'localStorage に書かれていない').not.toBeNull()
    return parse_kftl_tabs(raw!)
}

describe('useKftlTabs', () => {
    test('何度呼んでも同じインスタンスを返す', () => {
        expect(useKftlTabs()).toBe(useKftlTabs())
    })

    test('常に1枚以上のタブがある', () => {
        const store = useKftlTabs()
        expect(store.tabs.value.length).toBe(1)
        expect(store.last_active_tab_id.value).toBe(store.tabs.value[0].id)
    })

    test('本文の変更が localStorage に反映される', async () => {
        const store = useKftlTabs()
        store.set_tab_content(store.last_active_tab_id.value, 'メモ')
        await nextTick()

        expect(saved_tabs()!.tabs[0].content).toBe('メモ')
    })

    // setup の中で素に watch を張ると、そのコンポーネントの unmount で永続化ごと止まる。
    // ストアは独立した effectScope で作ってあるので、ここが壊れたら回帰
    test('最初にストアを取ったコンポーネントが unmount しても永続化が続く', async () => {
        const { app, store } = mount_store_host()
        app.unmount()

        store.set_tab_content(store.last_active_tab_id.value, 'unmount のあとに書いた')
        await nextTick()

        expect(saved_tabs()!.tabs[0].content).toBe('unmount のあとに書いた')
    })

    test('add_tab は末尾に足してアクティブにする', async () => {
        const store = useKftlTabs()
        const first_tab_id = store.last_active_tab_id.value

        const added_tab_id = store.add_tab('中身', 'テンプレ名')
        await nextTick()

        expect(store.tabs.value.map(tab => tab.id)).toEqual([first_tab_id, added_tab_id])
        expect(store.last_active_tab_id.value).toBe(added_tab_id)
        expect(store.get_tab_content(added_tab_id)).toBe('中身')
        expect(saved_tabs()!.active_tab_id).toBe(added_tab_id)
    })

    test('最後の1枚を閉じると空のタブが1枚できる', () => {
        const store = useKftlTabs()
        const first_tab_id = store.last_active_tab_id.value
        store.set_tab_content(first_tab_id, '消える')

        store.close_tab(first_tab_id)

        expect(store.tabs.value.length).toBe(1)
        expect(store.tabs.value[0].id).not.toBe(first_tab_id)
        expect(store.get_tab_content(store.last_active_tab_id.value)).toBe('')
    })

    // 「いま映しているタブ」はウィンドウごとなので、ここが持つのは
    // 「次に開くウィンドウの初期表示」だけ
    test('note_active_tab は実在するタブだけ受け付ける', () => {
        const store = useKftlTabs()
        const first_tab_id = store.last_active_tab_id.value
        const added_tab_id = store.add_tab('中身')

        store.note_active_tab('no-such-tab')
        expect(store.last_active_tab_id.value).toBe(added_tab_id)

        store.note_active_tab(first_tab_id)
        expect(store.last_active_tab_id.value).toBe(first_tab_id)
    })

    // 送信の往復中にそのタブが消えても落ちないようにする保険
    test('存在しないタブへの読み書きは落ちない', () => {
        const store = useKftlTabs()
        expect(store.get_tab_content('no-such-tab')).toBe('')
        expect(() => store.set_tab_content('no-such-tab', 'x')).not.toThrow()
        expect(() => store.note_active_tab('no-such-tab')).not.toThrow()
        expect(store.last_active_tab_id.value).toBe(store.tabs.value[0].id)
    })

    test('has_content は中身のあるタブが1枚でもあれば true', () => {
        const store = useKftlTabs()
        expect(store.has_content()).toBe(false)
        store.add_tab('  \n ')
        expect(store.has_content()).toBe(false)
        store.add_tab('メモ')
        expect(store.has_content()).toBe(true)
    })

    // プライベートモード等では localStorage が throw する。setup 中に落ちると KFTLView ごと死ぬ
    test('localStorage が使えない環境でも落ちない', async () => {
        vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('denied') })
        vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('denied') })

        const store = useKftlTabs()
        expect(store.tabs.value.length).toBe(1)
        store.set_tab_content(store.last_active_tab_id.value, 'メモ')
        await nextTick()
        expect(store.get_tab_content(store.last_active_tab_id.value)).toBe('メモ')
    })
})
