/**
 * メモ帳のタブの純関数テスト。
 *
 * タブ名の決め方・localStorage の往復・旧形式（単一キー kftl_content）からの移行を固定する。
 * 壊れた JSON で throw すると KFTLView の setup ごと落ちるので、
 * 「絶対に throw しない」ことも合わせて守る。
 */
import { beforeEach, describe, test, expect } from 'vitest'
import {
    add_kftl_tab,
    close_kftl_tab,
    create_kftl_tab,
    derive_kftl_tab_label,
    has_kftl_tab_content,
    load_kftl_tabs,
    parse_kftl_tabs,
    save_kftl_tabs,
    KFTL_LEGACY_CONTENT_STORAGE_KEY,
    KFTL_TABS_STORAGE_KEY,
    KFTL_TAB_LABEL_MAX_LENGTH,
} from '@/classes/kftl-tabs'

let id_counter = 0
function generate_id(): string {
    id_counter++
    return `tab-${id_counter}`
}

beforeEach(() => {
    localStorage.clear()
    id_counter = 0
})

describe('derive_kftl_tab_label', () => {
    test('テンプレート由来ならテンプレート名を出す', () => {
        const tab = create_kftl_tab('a', 'ーみ\n買い物', '買い物テンプレ')
        expect(derive_kftl_tab_label(tab, 0)).toBe('買い物テンプレ')
    })

    test('本文を書き換えてもテンプレート名は残る', () => {
        const tab = create_kftl_tab('a', 'まったく別の内容', '買い物テンプレ')
        expect(derive_kftl_tab_label(tab, 0)).toBe('買い物テンプレ')
    })

    test('テンプレート名が空白だけなら本文から取る', () => {
        const tab = create_kftl_tab('a', '本文の1行目', '   ')
        expect(derive_kftl_tab_label(tab, 0)).toBe('本文の1行目')
    })

    test('本文は最初の非空行を使う。前の空行は飛ばす', () => {
        const tab = create_kftl_tab('a', '\n\n  \n買い物メモ\n続き')
        expect(derive_kftl_tab_label(tab, 0)).toBe('買い物メモ')
    })

    test('長い行は切って省略記号を足す', () => {
        const long_line = 'あ'.repeat(KFTL_TAB_LABEL_MAX_LENGTH + 5)
        const tab = create_kftl_tab('a', long_line)
        expect(derive_kftl_tab_label(tab, 0)).toBe('あ'.repeat(KFTL_TAB_LABEL_MAX_LENGTH) + '…')
    })

    test('ちょうど上限の行は切らない', () => {
        const line = 'あ'.repeat(KFTL_TAB_LABEL_MAX_LENGTH)
        expect(derive_kftl_tab_label(create_kftl_tab('a', line), 0)).toBe(line)
    })

    test('中身が空なら通し番号', () => {
        expect(derive_kftl_tab_label(create_kftl_tab('a', ''), 0)).toBe('1')
        expect(derive_kftl_tab_label(create_kftl_tab('b', '  \n\n'), 2)).toBe('3')
    })
})

describe('add_kftl_tab / close_kftl_tab', () => {
    test('足したタブがアクティブになる', () => {
        const first = create_kftl_tab('a')
        const second = create_kftl_tab('b')
        const state = add_kftl_tab({ tabs: [first], active_tab_id: 'a' }, second)
        expect(state.tabs.map(tab => tab.id)).toEqual(['a', 'b'])
        expect(state.active_tab_id).toBe('b')
    })

    test('最後の1枚を閉じたら空のタブが1枚できる', () => {
        const state = close_kftl_tab({ tabs: [create_kftl_tab('a', '本文')], active_tab_id: 'a' }, 'a', generate_id)
        expect(state.tabs.length).toBe(1)
        expect(state.tabs[0].id).not.toBe('a')
        expect(state.tabs[0].content).toBe('')
        expect(state.active_tab_id).toBe(state.tabs[0].id)
    })

    test('アクティブなタブを閉じたら右隣へ移る', () => {
        const tabs = [create_kftl_tab('a'), create_kftl_tab('b'), create_kftl_tab('c')]
        const state = close_kftl_tab({ tabs: tabs, active_tab_id: 'b' }, 'b', generate_id)
        expect(state.tabs.map(tab => tab.id)).toEqual(['a', 'c'])
        expect(state.active_tab_id).toBe('c')
    })

    test('右隣が無ければ左隣へ移る', () => {
        const tabs = [create_kftl_tab('a'), create_kftl_tab('b')]
        const state = close_kftl_tab({ tabs: tabs, active_tab_id: 'b' }, 'b', generate_id)
        expect(state.active_tab_id).toBe('a')
    })

    test('アクティブでないタブを閉じてもアクティブは動かない', () => {
        const tabs = [create_kftl_tab('a'), create_kftl_tab('b'), create_kftl_tab('c')]
        const state = close_kftl_tab({ tabs: tabs, active_tab_id: 'c' }, 'a', generate_id)
        expect(state.tabs.map(tab => tab.id)).toEqual(['b', 'c'])
        expect(state.active_tab_id).toBe('c')
    })

    test('無いタブを閉じようとしたら何もしない', () => {
        const before = { tabs: [create_kftl_tab('a')], active_tab_id: 'a' }
        expect(close_kftl_tab(before, 'zzz', generate_id)).toBe(before)
    })
})

describe('has_kftl_tab_content', () => {
    test('1枚でも中身があれば true', () => {
        expect(has_kftl_tab_content([create_kftl_tab('a', '  '), create_kftl_tab('b', 'メモ')])).toBe(true)
    })

    test('全部が空白だけなら false', () => {
        expect(has_kftl_tab_content([create_kftl_tab('a', ''), create_kftl_tab('b', ' \n ')])).toBe(false)
    })
})

describe('parse_kftl_tabs', () => {
    test.each([
        ['壊れたJSON', '{'],
        ['空文字', ''],
        ['null', 'null'],
        ['配列でない tabs', '{"tabs":"x"}'],
        ['tabs が空', '{"tabs":[]}'],
        ['id の無いタブだけ', '{"tabs":[{"content":"a"}]}'],
    ])('%s は throw せず null を返す', (_name, raw) => {
        expect(parse_kftl_tabs(raw)).toBeNull()
    })

    test('欠けた項目は既定値で埋める', () => {
        const state = parse_kftl_tabs('{"tabs":[{"id":"a"}],"active_tab_id":"a"}')
        expect(state).not.toBeNull()
        expect(state!.tabs[0].content).toBe('')
        expect(state!.tabs[0].template_name).toBeNull()
    })

    test('active_tab_id が実在しなければ先頭に倒す', () => {
        const state = parse_kftl_tabs('{"tabs":[{"id":"a"},{"id":"b"}],"active_tab_id":"zzz"}')
        expect(state!.active_tab_id).toBe('a')
    })
})

describe('load_kftl_tabs / save_kftl_tabs', () => {
    test('保存して読み直すと同じ内容に戻る', () => {
        const tabs = [create_kftl_tab('a', 'ひとつめ'), create_kftl_tab('b', 'ふたつめ', 'テンプレ')]
        save_kftl_tabs({ tabs: tabs, active_tab_id: 'b' })

        const loaded = load_kftl_tabs(generate_id)
        expect(loaded.tabs).toEqual(tabs)
        expect(loaded.active_tab_id).toBe('b')
    })

    test('何も保存されていなければ空のタブが1枚', () => {
        const loaded = load_kftl_tabs(generate_id)
        expect(loaded.tabs.length).toBe(1)
        expect(loaded.tabs[0].content).toBe('')
        expect(loaded.active_tab_id).toBe(loaded.tabs[0].id)
    })

    test('壊れたJSONが入っていても空のタブが1枚', () => {
        localStorage.setItem(KFTL_TABS_STORAGE_KEY, '{')
        const loaded = load_kftl_tabs(generate_id)
        expect(loaded.tabs.length).toBe(1)
        expect(loaded.tabs[0].content).toBe('')
    })

    test('旧形式の下書きはタブ1枚へ移行し、旧キーを消して書き戻す', () => {
        localStorage.setItem(KFTL_LEGACY_CONTENT_STORAGE_KEY, '移行される下書き')

        const loaded = load_kftl_tabs(generate_id)

        expect(loaded.tabs.length).toBe(1)
        expect(loaded.tabs[0].content).toBe('移行される下書き')
        expect(localStorage.getItem(KFTL_LEGACY_CONTENT_STORAGE_KEY), '旧キーが残っている').toBeNull()
        // 何も入力せずリロードしても消えないよう、その場で新形式へ書き戻す
        expect(load_kftl_tabs(generate_id).tabs[0].content).toBe('移行される下書き')
    })

    test('新形式があれば旧キーは見ない', () => {
        save_kftl_tabs({ tabs: [create_kftl_tab('a', '新形式')], active_tab_id: 'a' })
        localStorage.setItem(KFTL_LEGACY_CONTENT_STORAGE_KEY, '旧形式')

        expect(load_kftl_tabs(generate_id).tabs[0].content).toBe('新形式')
    })
})
