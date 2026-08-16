import { effectScope, ref, watch, type EffectScope, type Ref } from 'vue'
import { GkillAPI } from '@/classes/api/gkill-api'
import {
    add_kftl_tab,
    close_kftl_tab,
    create_kftl_tab,
    has_kftl_tab_content,
    load_kftl_tabs,
    save_kftl_tabs,
    type KFTLTabState,
} from '@/classes/kftl-tabs'

/**
 * メモ帳のタブを持つ共有ストア。
 *
 * **モジュールシングルトンなのは意図的。** KFTLView は `/kftl` ページ・各画面の
 * メモ帳ダイアログ・打刻メモ帳の3系統から使われ、メモ帳ダイアログは複数枚開ける。
 * インスタンスごとに配列を持つと、片方の古い配列で localStorage を丸ごと上書きして
 * もう片方のタブが消える（タブ化する前は単一文字列だったので「上書き」で済んでいた）。
 * 真実を1つにすれば、この競合が構造的に起きない。
 *
 * **「いま映しているタブ」はここには置かない。** ウィンドウごとに別のタブを選べるように、
 * アクティブタブは `useKftlView` が各自で持つ。ここが持つのは
 * 「次に開くウィンドウが最初に映すタブ」＝ `last_active_tab_id` だけ。
 */
export interface KFTLTabsStore {
    tabs: Ref<Array<KFTLTabState>>

    /** 直近にどこかのウィンドウで選ばれたタブ。新しいウィンドウの初期表示と永続化に使う */
    last_active_tab_id: Ref<string>

    has_tab(tab_id: string): boolean
    get_tab_content(tab_id: string): string
    set_tab_content(tab_id: string, content: string): void

    add_tab(content?: string, template_name?: string | null): string
    close_tab(tab_id: string): void

    /** ウィンドウがタブを切り替えたことを知らせる（次に開くウィンドウの初期値になる） */
    note_active_tab(tab_id: string): void

    has_content(): boolean

    /**
     * 送信中のタブを掴む。掴めたら true、false なら別のウィンドウが送信中。
     *
     * `is_requested_submit` / `is_submitting` は**ビューごと**なので、同じタブを映している
     * 別ウィンドウの保存は止められない。KFTLは複数リクエストをtxで束ねて送るため、
     * 二重送信するとKyouが丸ごと重複登録される。ウィンドウをまたいだ排他はここにしか置けない
     */
    try_begin_submit(tab_id: string): boolean

    /** 送信中のタブを手放す。冪等（掴んでいないタブの解放は no-op） */
    end_submit(tab_id: string): void
}

let shared_store: KFTLTabsStore | null = null
let shared_scope: EffectScope | null = null

export function useKftlTabs(): KFTLTabsStore {
    if (shared_store === null) {
        // 独立した effectScope で作る。setup の中で素に watch を張ると、
        // 最初に呼んだコンポーネントのスコープに属してしまい、そのコンポーネントが
        // unmount された時点で**永続化の watch ごと止まる**（メモ帳ダイアログを閉じる、
        // 画面を移る、で再現する）
        shared_scope = effectScope(true)
        shared_store = shared_scope.run(() => create_kftl_tabs_store())!
    }
    return shared_store
}

/** 単体テスト用。シングルトンなのでテスト間に状態が漏れる */
export function reset_kftl_tabs_for_test(): void {
    shared_scope?.stop()
    shared_scope = null
    shared_store = null
}

function generate_tab_id(): string {
    return GkillAPI.get_instance().generate_uuid()
}

function create_kftl_tabs_store(): KFTLTabsStore {
    const loaded = load_kftl_tabs(generate_tab_id)

    const tabs: Ref<Array<KFTLTabState>> = ref(loaded.tabs)
    const last_active_tab_id: Ref<string> = ref(loaded.active_tab_id)

    // ストアはコンポーネントのスコープ外に生きるので、この watch は止めない（止めると永続化が死ぬ）
    watch([tabs, last_active_tab_id], () => {
        save_kftl_tabs({ tabs: tabs.value, active_tab_id: last_active_tab_id.value })
    }, { deep: true })

    function find_tab(tab_id: string): KFTLTabState | null {
        return tabs.value.find(tab => tab.id === tab_id) ?? null
    }

    function has_tab(tab_id: string): boolean {
        return find_tab(tab_id) !== null
    }

    function get_tab_content(tab_id: string): string {
        return find_tab(tab_id)?.content ?? ""
    }

    // 存在しないタブへの書き込みは no-op。送信の往復中にそのタブが消えても落ちないようにする
    function set_tab_content(tab_id: string, content: string): void {
        const tab = find_tab(tab_id)
        if (tab === null) {
            return
        }
        tab.content = content
    }

    function add_tab(content: string = "", template_name: string | null = null): string {
        const tab = create_kftl_tab(generate_tab_id(), content, template_name)
        const next = add_kftl_tab({ tabs: tabs.value, active_tab_id: last_active_tab_id.value }, tab)
        tabs.value = next.tabs
        last_active_tab_id.value = next.active_tab_id
        return tab.id
    }

    function close_tab(tab_id: string): void {
        const next = close_kftl_tab({ tabs: tabs.value, active_tab_id: last_active_tab_id.value }, tab_id, generate_tab_id)
        tabs.value = next.tabs
        last_active_tab_id.value = next.active_tab_id
    }

    function note_active_tab(tab_id: string): void {
        if (!has_tab(tab_id)) {
            return
        }
        last_active_tab_id.value = tab_id
    }

    function has_content(): boolean {
        return has_kftl_tab_content(tabs.value)
    }

    /**
     * 送信中のタブ。
     *
     * 素の Set で持つ（リアクティブにしない）。UIからは読まないので ref にする必要が無く、
     * `is_tab_locked` に混ぜるのは**禁止** ―― 「送信中のタブから他のタブへ切り替えられない」
     * という別の不具合になる。
     * localStorage にも出さない（永続化 watch は [tabs, last_active_tab_id] だけを見ている）。
     * リロードで掴んだままのタブが二度と保存できなくなるため、出してはいけない
     */
    const submitting_tab_ids = new Set<string>()

    function try_begin_submit(tab_id: string): boolean {
        if (submitting_tab_ids.has(tab_id)) {
            return false
        }
        submitting_tab_ids.add(tab_id)
        return true
    }

    function end_submit(tab_id: string): void {
        submitting_tab_ids.delete(tab_id)
    }

    return {
        tabs,
        last_active_tab_id,
        has_tab,
        get_tab_content,
        set_tab_content,
        add_tab,
        close_tab,
        note_active_tab,
        has_content,
        try_begin_submit,
        end_submit,
    }
}
