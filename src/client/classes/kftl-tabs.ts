'use strict'

/**
 * KFTL（メモ帳）のタブ状態。
 *
 * ここには純関数だけを置く。localStorage の読み書きも含めて副作用は
 * この1ファイルに閉じ込め、ストア（use-kftl-tabs.ts）は組み立て済みの状態を受け取るだけにする。
 * id の採番だけは呼び出し元（GkillAPI.generate_uuid）から注入する。
 */

export interface KFTLTabState {
    id: string

    content: string

    /** テンプレートから開いたタブならそのテンプレート名（KFTLTemplateElementData.title）。それ以外は null */
    template_name: string | null
}

export interface KFTLTabsState {
    tabs: Array<KFTLTabState>
    active_tab_id: string
}

/**
 * タブを離れたときの textarea の状態。戻ってきたときに復元する。
 * 永続化はしない（保存量が増えるだけで、リロード後に復元する意味が薄い）
 */
export interface KFTLEditorViewState {
    selection_start: number
    selection_end: number
    scroll_top: number
}

export const KFTL_TABS_STORAGE_KEY = "kftl_tabs"

/** タブ化する前の単一下書きのキー。初回起動時にタブ1枚へ移行して消す */
export const KFTL_LEGACY_CONTENT_STORAGE_KEY = "kftl_content"

const KFTL_TABS_STORAGE_VERSION = 1

/** タブ見出しに出す最大文字数。超えたら省略記号を足す */
export const KFTL_TAB_LABEL_MAX_LENGTH = 12

export function create_kftl_tab(id: string, content: string = "", template_name: string | null = null): KFTLTabState {
    return {
        id: id,
        content: content,
        template_name: template_name,
    }
}

/**
 * タブ見出し。テンプレート名 → 本文の最初の非空行 → 通し番号 の順で決める。
 *
 * テンプレート名は本文を書き換えても持ち続ける。テンプレートの1行目は
 * 「ーみ」のようなプレフィックス行のことが多く、本文から取ると見分けがつかないため。
 * 番号が出るのは中身が空のタブだけで、打ち始めた時点で1行目に置き換わる。
 */
export function derive_kftl_tab_label(tab: KFTLTabState, index: number): string {
    if (tab.template_name !== null) {
        const template_name = tab.template_name.trim()
        if (template_name !== "") {
            return truncate_kftl_tab_label(template_name)
        }
    }

    const lines = tab.content.split("\n")
    for (let i = 0; i < lines.length; i++) {
        const line = lines[i].trim()
        if (line !== "") {
            return truncate_kftl_tab_label(line)
        }
    }

    return String(index + 1)
}

function truncate_kftl_tab_label(label: string): string {
    if (label.length <= KFTL_TAB_LABEL_MAX_LENGTH) {
        return label
    }
    return label.slice(0, KFTL_TAB_LABEL_MAX_LENGTH) + "…"
}

/** タブを1枚足して、それをアクティブにする */
export function add_kftl_tab(state: KFTLTabsState, tab: KFTLTabState): KFTLTabsState {
    return {
        tabs: [...state.tabs, tab],
        active_tab_id: tab.id,
    }
}

/**
 * タブを1枚閉じる。
 *
 * 閉じた結果0枚になるなら空のタブを1枚作る（タブは常に1枚以上）。
 * 閉じたのがアクティブタブだったときの行き先は右隣、無ければ左隣。
 */
export function close_kftl_tab(state: KFTLTabsState, tab_id: string, generate_id: () => string): KFTLTabsState {
    const index = state.tabs.findIndex(tab => tab.id === tab_id)
    if (index === -1) {
        return state
    }

    const tabs = state.tabs.filter(tab => tab.id !== tab_id)
    if (tabs.length === 0) {
        const new_tab = create_kftl_tab(generate_id())
        return { tabs: [new_tab], active_tab_id: new_tab.id }
    }

    if (state.active_tab_id !== tab_id) {
        return { tabs: tabs, active_tab_id: state.active_tab_id }
    }
    const next_index = Math.min(index, tabs.length - 1)
    return { tabs: tabs, active_tab_id: tabs[next_index].id }
}

/** タブのうち1枚でも中身があるか。ページ離脱の警告に使う */
export function has_kftl_tab_content(tabs: ReadonlyArray<KFTLTabState>): boolean {
    return tabs.some(tab => tab.content.trim() !== "")
}

/**
 * 保存済みのタブを読む。
 *
 * 旧形式（単一キー kftl_content）しか無ければタブ1枚へ移行し、旧キーを消して
 * その場で新形式を書き戻す（書き戻さないと、利用者が何も入力せずリロードしただけで
 * 下書きが消える）。壊れたJSONは空タブ1枚に倒す。
 */
export function load_kftl_tabs(generate_id: () => string): KFTLTabsState {
    const raw = read_local_storage(KFTL_TABS_STORAGE_KEY)
    if (raw !== null) {
        const parsed = parse_kftl_tabs(raw)
        if (parsed !== null) {
            return parsed
        }
    }

    const legacy_content = read_local_storage(KFTL_LEGACY_CONTENT_STORAGE_KEY)
    if (legacy_content !== null) {
        remove_local_storage(KFTL_LEGACY_CONTENT_STORAGE_KEY)
        const tab = create_kftl_tab(generate_id(), legacy_content)
        const state = { tabs: [tab], active_tab_id: tab.id }
        save_kftl_tabs(state)
        return state
    }

    const tab = create_kftl_tab(generate_id())
    return { tabs: [tab], active_tab_id: tab.id }
}

export function save_kftl_tabs(state: KFTLTabsState): void {
    const serialized = JSON.stringify({
        version: KFTL_TABS_STORAGE_VERSION,
        tabs: state.tabs,
        active_tab_id: state.active_tab_id,
    })
    write_local_storage(KFTL_TABS_STORAGE_KEY, serialized)
}

/** 壊れていたら null。呼び出し元が旧形式の移行か空タブ1枚へ倒す。絶対に throw しない */
export function parse_kftl_tabs(raw: string): KFTLTabsState | null {
    let parsed: unknown = null
    try {
        parsed = JSON.parse(raw)
    } catch (_err: unknown) {
        return null
    }
    if (parsed === null || typeof parsed !== 'object') {
        return null
    }

    const record = parsed as Record<string, unknown>
    const raw_tabs = record['tabs']
    if (!Array.isArray(raw_tabs)) {
        return null
    }

    const tabs = new Array<KFTLTabState>()
    for (let i = 0; i < raw_tabs.length; i++) {
        const tab = normalize_kftl_tab(raw_tabs[i])
        if (tab !== null) {
            tabs.push(tab)
        }
    }
    if (tabs.length === 0) {
        return null
    }

    const raw_active_tab_id = record['active_tab_id']
    const active_tab_id = (typeof raw_active_tab_id === 'string' && tabs.some(tab => tab.id === raw_active_tab_id))
        ? raw_active_tab_id
        : tabs[0].id
    return { tabs: tabs, active_tab_id: active_tab_id }
}

/** id さえ読めれば拾う。欠けている項目は既定値で埋める */
function normalize_kftl_tab(raw: unknown): KFTLTabState | null {
    if (raw === null || typeof raw !== 'object') {
        return null
    }
    const record = raw as Record<string, unknown>
    const id = record['id']
    if (typeof id !== 'string' || id === "") {
        return null
    }
    const content = record['content']
    const template_name = record['template_name']
    return {
        id: id,
        content: typeof content === 'string' ? content : "",
        template_name: typeof template_name === 'string' ? template_name : null,
    }
}

// localStorage は Safari のプライベートモード等で throw しうる。
// setup 中に throw すると KFTLView ごと落ちるので、読み書きはすべてここを通す。
function read_local_storage(key: string): string | null {
    try {
        return localStorage.getItem(key)
    } catch (_err: unknown) {
        return null
    }
}

function write_local_storage(key: string, value: string): void {
    try {
        localStorage.setItem(key, value)
    } catch (_err: unknown) {
        // 保存できない環境ではタブの永続化を諦める
    }
}

function remove_local_storage(key: string): void {
    try {
        localStorage.removeItem(key)
    } catch (_err: unknown) {
        // 同上
    }
}
