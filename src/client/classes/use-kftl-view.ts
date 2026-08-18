import { computed, nextTick, onMounted, onUnmounted, ref, useId, watch, type Ref } from 'vue'
import { i18n } from '@/i18n'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillMessage } from '@/classes/api/gkill-message'
import { LineLabelData } from '@/classes/kftl/line-label-data'
import { KFTLStatement } from '@/classes/kftl/kftl-statement'
import { KFTL_ASCII_SAVE_CHARACTOR } from '@/classes/kftl/kftl-prefixes'
import { TextAreaInfo } from '@/classes/kftl/text-area-info'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { DiscardTXRequest } from '@/classes/api/req_res/discard-tx-request'
import { CommitTXRequest } from '@/classes/api/req_res/commit-tx-request'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import type { KFTLProps } from '@/pages/views/kftl-props'
import type { KFTLViewEmits } from '@/pages/views/kftl-view-emits'
import type { KFTLRequest, KFTLRequestResult } from '@/classes/kftl/kftl-request'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import type { ComponentRef } from '@/classes/component-ref'
import { useConfirmUnknownMiBoard } from '@/classes/use-confirm-unknown-mi-board'
import { useKftlTabs } from '@/classes/use-kftl-tabs'
import { derive_kftl_tab_label, type KFTLEditorViewState, type KFTLTabState } from '@/classes/kftl-tabs'

/**
 * textarea の id の接頭辞。
 *
 * メモ帳ダイアログは複数枚開けるので、id を固定すると document 内で重複する。
 * 内部のロジックはもう id を引かない（テンプレート ref を使う）が、
 * 重複した id は Playwright の strict mode に引っかかる。
 * E2E は class の `.kftl_text_area` で掴むこと
 */
export const KFTL_TEXT_AREA_ELEMENT_ID_PREFIX = "kftl_text_area"

export function useKftlView(options: {
    props: KFTLProps,
    emits: KFTLViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kftl_template_dialog = ref<ComponentRef | null>(null)
    const confirm_close_kftl_tab_dialog = ref<ComponentRef | null>(null)
    const kftl_text_area = ref<HTMLTextAreaElement | null>(null)
    const kftl_line_label_wrap = ref<HTMLElement | null>(null)

    // 複数枚のメモ帳ウィンドウで重複しないよう、インスタンスごとに採番する
    const text_area_element_id = `${KFTL_TEXT_AREA_ELEMENT_ID_PREFIX}-${useId()}`

    // ── Confirm unknown mi board ──
    const confirm_unknown_mi_board = useConfirmUnknownMiBoard({ application_config: () => props.application_config })

    // ── Confirm unknown tag ──
    const confirm_unknown_tag = useConfirmUnknownTag({ application_config: () => props.application_config })

    // ── Tabs ──
    // タブの一覧と中身は共有シングルトン（use-kftl-tabs.ts のコメント参照）。
    // メモ帳ダイアログは複数枚開けるので、インスタンスごとに配列を持つと
    // 片方の古い配列でもう片方のタブを消してしまう。
    //
    // 一方「いま映しているタブ」はこのビューだけのもの。ウィンドウごとに別の下書きを
    // 開いて並べられるようにするため、ここで持つ
    const tabs_store = useKftlTabs()
    const tabs = tabs_store.tabs
    const active_tab_id: Ref<string> = ref(tabs_store.last_active_tab_id.value)

    // キャレットとスクロールもビューごと。同じタブを2枚のウィンドウで開いても奪い合わない
    const editor_view_states = new Map<string, KFTLEditorViewState>()

    /**
     * 確認ダイアログをまたいで持ち越す送信対象タブ。
     *
     * 送信は未知タグ確認・未知板名確認でいったん `do_submit` を抜けて応答を待つ。
     * gkill のフローティングダイアログは非モーダル（App.vue の `.gkill-float-scrim` が
     * `pointer-events: none`）なので、確認中でも背後のタブバーは押せてしまう。
     *
     * **`do_submit` には引数で渡す。** ここを見に行くのは確認からの続行（`confirm_submit` /
     * `confirm_mi_board_submit`）だけなので、ダイアログを Escape やブラウザバックで
     * 閉じられて古い値が残っても、次の送信が別のタブへ誤配送されることはない
     */
    const submit_target_tab_id: Ref<string | null> = ref(null)

    /**
     * 送信が飛んでいる間だけ真。
     *
     * タブ操作のロックに `is_requested_submit` は使えない ―― あれは
     * `application_config` の読み込みが終わるまで真なので、起動直後にタブを足せなくなる
     */
    const is_submitting: Ref<boolean> = ref(false)

    /** 内容が残っているタブを × したときの確認待ち */
    const pending_close_tab_id: Ref<string | null> = ref(null)

    // ── State refs ──
    const text_area_width: Ref<number | 'unset'> = ref(0)
    const text_area_height: Ref<number> = ref(0)
    const line_label_width: Ref<number> = ref(0)
    const line_label_height: Ref<number> = ref(0)

    // v-tabs の既定高さは48px。テンプレート側で :height に渡して、タイトル行に収める。
    // 測った値をフィードバックすると kftl-dialog.vue の ResizeObserver が縮小ループに入る
    const tab_bar_height = 40
    // タブ列はタイトル行に同居しているので、引くのはタイトル行のぶんだけ。
    // タブバー＋上下2pxずつの余白ちょうどにする ―― 大きくすると
    // タイトルとテキストエリアのあいだに何も無い帯ができる。
    // 実寸は kftl-view.vue の .kftl_title に CSS で固定してある（v-card-title に height prop は無い）
    const title_height = tab_bar_height + 4
    const action_height = 10
    // /mkfl は app_content_height を半分にして渡してくる。小さい画面で負値にしない
    const min_text_area_height = 80
    const kftl_input_height: Ref<number> = ref(0)
    const kftl_input_width: Ref<number | 'unset'> = ref(0)

    const line_label_datas: Ref<Array<LineLabelData>> = ref(new Array<LineLabelData>())
    const line_label_styles: Ref<Array<Record<string, string>>> = ref(new Array<Record<string, string>>())
    const invalid_line_numbers: Ref<Array<number>> = ref(new Array<number>())
    const is_requested_submit: Ref<boolean> = ref(true)
    /**
     * タグ確認が開いているか。
     *
     * 共有ダイアログ(ConfirmUnknownTagDialog)は自分で表示状態を持つので、
     * こちらは「開いた」で立て、`closed` イベントで倒すだけの写し。
     * `unknown_tags` の空判定で代用してはいけない ―― ブラウザバックで閉じても
     * 空にならないので、タブ操作が永久にロックされる
     */
    const is_confirm_unknown_tag_open: Ref<boolean> = ref(false)

    // ── Computed ──
    const text_area_width_px = computed(() => text_area_width.value.toString().concat("px"))
    const text_area_height_px = computed(() => text_area_height.value.toString().concat("px"))
    const line_label_width_px = computed(() => line_label_width.value.toString().concat("px"))
    const line_label_height_px = computed(() => line_label_height.value.toString().concat("px"))
    const kftl_input_height_px = computed(() => kftl_input_height.value.toString().concat("px"))
    const kftl_input_width_px = computed(() => kftl_input_width.value.toString().concat("px"))
    const title_height_px = computed(() => title_height.toString().concat("px"))

    /**
     * タブの切替・追加・クローズを止める条件。
     *
     * タグ確認が開いている間も止めるが、これは `is_confirm_unknown_tag_open` が
     * ダイアログの `closed`（どの閉じ方でも1回だけ来る）で倒れるので固まらない。
     * 板名確認はブラウザバックで閉じても `unknown_mi_boards` が空にならないため、
     * ロック条件に入れていない（入れると永久ロックになる）
     */
    const is_tab_locked = computed(() => is_submitting.value || is_confirm_unknown_tag_open.value)

    /** アクティブなタブの本文。setter があるので v-model にそのまま渡せる */
    const text_area_content = computed<string>({
        get: () => tabs_store.get_tab_content(active_tab_id.value),
        set: (value: string) => tabs_store.set_tab_content(active_tab_id.value, value),
    })

    /** v-tabs の v-model。切替のたびにキャレットとスクロールを退避・復元したいので computed を挟む */
    const active_tab_id_model = computed<string>({
        get: () => active_tab_id.value,
        set: (value: string) => activate_tab(value),
    })

    const pending_close_tab_label = computed(() => {
        const tab_id = pending_close_tab_id.value
        if (tab_id === null) {
            return ""
        }
        const index = tabs.value.findIndex(tab => tab.id === tab_id)
        if (index === -1) {
            return ""
        }
        return derive_kftl_tab_label(tabs.value[index], index)
    })

    function tab_label(tab: KFTLTabState, index: number): string {
        return derive_kftl_tab_label(tab, index)
    }

    // ── Watchers ──
    if (props.application_config.is_loaded) {
        is_requested_submit.value = false
    }
    watch(() => props.application_config, () => {
        if (props.application_config.is_loaded) {
            is_requested_submit.value = false
        }
    })

    /**
     * 保存マーカーによる自動送信を「利用者が打ったとき」だけに限るための印。
     *
     * この判定を watch の内容変化そのものに戻すと、タブ切替・localStorage からの復元でも
     * `text_area_content` が変わるので発火してしまい、末尾にマーカーが残ったタブを
     * クリックしただけで保存が走る（設定の読み込み前はマーカー付きのまま保存されうる）。
     *
     * 立てるのは `onTextAreaInput()` **だけ**。テンプレート貼り付けはこの印に相乗りさせず、
     * `paste_template` から `maybe_submit_by_save_marker()` を直接呼ぶ（理由はそちらのコメント）
     */
    let user_input_tab_id: string | null = null

    // flush: 'post' なのは、コールバックの中で textarea の実寸とスクロール位置を読むため
    watch(() => text_area_content.value, async (new_value, old_value) => {
        if (new_value === old_value) {
            return
        }
        const input_tab_id = user_input_tab_id
        user_input_tab_id = null

        // **保存マーカーの判定は「この変更で確定した本文」で行い、解析を待たない。**
        // 行ラベルと不正行の再計算は await を挟む（`get_invalid_line_indexs` は
        // 行ごとに await するので行数に比例して伸びる）。待ってから
        // `text_area_content.value` を読み直すと、その間に1文字打たれただけで
        // 末尾がマーカーでなくなり、**保存が黙って起きない**。
        // 行数が多いタブほど窓が広がるので「たまに効かない」ように見える
        if (input_tab_id !== null && input_tab_id === active_tab_id.value) {
            await maybe_submit_by_save_marker(new_value, old_value)
        }

        update_line_labels()
        await refresh_invalid_lines()
    }, { flush: 'post' })

    /**
     * 消えたタブへの追随。
     *
     * タブは共有なので、**別のウィンドウが閉じたり、保存でタブが閉じたり**すると、
     * このウィンドウが映していたタブが無くなる。放っておくと `text_area_content` が
     * 空文字を返し続ける（存在しないタブへの読み書きは no-op）ので、
     * 消える前の位置へクランプして隣のタブへ移る。
     */
    watch(() => tabs.value.map(tab => tab.id), (new_tab_ids, old_tab_ids) => {
        if (new_tab_ids.includes(active_tab_id.value)) {
            return
        }
        if (new_tab_ids.length === 0) {
            return
        }
        const old_index = old_tab_ids ? old_tab_ids.indexOf(active_tab_id.value) : -1
        const next_index = old_index === -1 ? 0 : Math.min(old_index, new_tab_ids.length - 1)
        active_tab_id.value = new_tab_ids[Math.max(0, next_index)]
    })

    watch(line_label_datas, async () => {
        line_label_styles.value.splice(0)
        let prev_target_id = ""
        let background_is_gray = true
        let switch_id = false
        let background_color: string = "white"
        for (let i = 0; i < line_label_datas.value.length; i++) {
            let color: string = "unset"
            switch_id = prev_target_id != line_label_datas.value[i].target_request_id
            if (switch_id) {
                background_is_gray = !background_is_gray
                if (background_is_gray) {
                    if (props.application_config.use_dark_theme) {
                        background_color = '#C0C0C0'
                    } else {
                        background_color = "#f0f0f0"
                    }
                } else {
                    background_color = ""
                }
            }
            if (is_invalid_line(i)) {
                color = "pink"
            }
            line_label_styles.value.push({
                "background-color": background_color,
                "color": color,
            })
            prev_target_id = line_label_datas.value[i].target_request_id
        }
    })

    // ── Lifecycle ──
    function onResize() {
        resize()
        update_line_labels()
    }
    window.addEventListener("resize", onResize)
    watch(() => [props.app_content_width, props.app_content_height], () => {
        resize()
        update_line_labels()
    })

    // ── beforeunload guard ──
    // どれかのタブに未保存の内容がある場合、ページ離脱時に警告を表示する
    function onBeforeunload(e: BeforeUnloadEvent) {
        if (tabs_store.has_content()) {
            e.preventDefault()
        }
    }
    window.addEventListener("beforeunload", onBeforeunload)

    onMounted(async () => {
        resize()
        await nextTick()
        update_line_labels()
        await refresh_invalid_lines()
    })
    onUnmounted(() => {
        window.removeEventListener("resize", onResize)
        window.removeEventListener("beforeunload", onBeforeunload)
    })

    // ── Internal helpers ──
    function sync_line_label_scroll(text_area_element: HTMLElement): void {
        kftl_line_label_wrap.value?.scrollTo(0, text_area_element.scrollTop)
    }

    /**
     * 行ラベルの再生成。textarea の実寸を測るので DOM が要る。
     *
     * 世代トークンを持つのは、await を挟むあいだにタブが切り替わると
     * 前のタブぶんのラベルが後から着地するため
     */
    let line_label_generation = 0
    async function update_line_labels(): Promise<void> {
        const generation = ++line_label_generation
        const text_area_element = kftl_text_area.value
        if (text_area_element === null) {
            return
        }

        sync_line_label_scroll(text_area_element)

        const statement = new KFTLStatement(text_area_content.value)
        const textarea_info = new TextAreaInfo()
        textarea_info.text_area_element = text_area_element
        textarea_info.text_area_element_id = text_area_element_id

        const label_datas = statement.generate_line_label_data(textarea_info)
        if (generation !== line_label_generation) {
            return
        }
        line_label_datas.value = label_datas

        // ラベル再生成後、DOM更新を待ってスクロール位置を再同期する
        await nextTick()
        if (generation !== line_label_generation) {
            return
        }
        sync_line_label_scroll(text_area_element)
    }

    /**
     * 不正な行の洗い出し。DOM に依存しないので、textarea がまだ無くても動く。
     *
     * `do_submit` の先頭でこの結果を見て中断するため、タブを切り替えた直後に
     * 前のタブぶんの結果が着地すると「おかしな行があります」で保存できなくなる。
     * 世代トークンで最後の1回だけを書き戻す
     */
    let invalid_line_generation = 0
    async function refresh_invalid_lines(): Promise<void> {
        const generation = ++invalid_line_generation
        const statement = new KFTLStatement(text_area_content.value)
        const invalid_lines = await statement.get_invalid_line_indexs()
        if (generation !== invalid_line_generation) {
            return
        }
        invalid_line_numbers.value = invalid_lines
    }

    function is_invalid_line(line_index: number): boolean {
        for (let i = 0; i < invalid_line_numbers.value.length; i++) {
            if (invalid_line_numbers.value[i] == line_index) {
                return true
            }
        }
        return false
    }

    async function resize(): Promise<void> {
        const content_height = Math.max(
            min_text_area_height,
            props.app_content_height.valueOf() - title_height - action_height,
        )
        line_label_width.value = 100
        line_label_height.value = content_height
        text_area_width.value = props.app_content_width.valueOf() - line_label_width.value.valueOf() - 7 // 7はマジックナンバー
        text_area_height.value = content_height
        kftl_input_width.value = line_label_width.value.valueOf() + text_area_width.value.valueOf()
        kftl_input_height.value = content_height
    }

    // ── Editor view state（キャレットとスクロール。永続化はしない） ──
    function capture_editor_view_state(): void {
        const text_area_element = kftl_text_area.value
        if (text_area_element === null) {
            return
        }
        editor_view_states.set(active_tab_id.value, {
            selection_start: text_area_element.selectionStart,
            selection_end: text_area_element.selectionEnd,
            scroll_top: text_area_element.scrollTop,
        })
    }

    async function focus_active_tab_editor(): Promise<void> {
        await nextTick()
        const text_area_element = kftl_text_area.value
        if (text_area_element === null) {
            return
        }
        text_area_element.focus()
        const view_state = editor_view_states.get(active_tab_id.value) ?? null
        if (view_state === null) {
            const caret = text_area_element.value.length
            text_area_element.setSelectionRange(caret, caret)
            text_area_element.scrollTop = 0
        } else {
            text_area_element.setSelectionRange(view_state.selection_start, view_state.selection_end)
            text_area_element.scrollTop = view_state.scroll_top
        }
        sync_line_label_scroll(text_area_element)
    }

    // ── Tab operations ──
    function add_tab(): void {
        if (is_tab_locked.value) {
            return
        }
        capture_editor_view_state()
        active_tab_id.value = tabs_store.add_tab()
        focus_active_tab_editor()
    }

    function activate_tab(tab_id: string): void {
        if (is_tab_locked.value) {
            return
        }
        if (tab_id === active_tab_id.value) {
            return
        }
        if (!tabs_store.has_tab(tab_id)) {
            return
        }
        capture_editor_view_state()
        active_tab_id.value = tab_id
        tabs_store.note_active_tab(tab_id)
        focus_active_tab_editor()
    }

    /** 中身が残っているタブだけ確認を挟む。空のタブは黙って閉じる */
    function request_close_tab(tab_id: string): void {
        if (is_tab_locked.value) {
            return
        }
        if (tabs_store.get_tab_content(tab_id).trim() === "") {
            close_tab(tab_id)
            return
        }
        pending_close_tab_id.value = tab_id
        confirm_close_kftl_tab_dialog.value?.show()
    }

    function confirm_close_tab(): void {
        const tab_id = pending_close_tab_id.value
        pending_close_tab_id.value = null
        if (tab_id === null) {
            return
        }
        close_tab(tab_id)
    }

    function cancel_close_tab(): void {
        pending_close_tab_id.value = null
    }

    function close_tab(tab_id: string): void {
        capture_editor_view_state()
        editor_view_states.delete(tab_id)
        // アクティブなタブを閉じたときの行き先は、下の「消えたタブへの追随」が決める
        tabs_store.close_tab(tab_id)
        focus_active_tab_editor()
    }

    // ── Business logic ──
    /**
     * 確定した保存マーカー行の数。
     *
     * 「確定した」= その行の後ろに改行がある、という意味。`！` を打った時点ではまだ数えず、
     * 改行を打って初めて1になる(そうしないと打った瞬間に保存が走る)。
     * 行そのものが全角「！」または半角「!」だけであることを要求するので、
     * 1行目のマーカーも、末尾以外にあるマーカーも同じ規則で数えられる。
     */
    function count_save_marker_lines(text: string): number {
        const ja_marker = i18n.global.t("KFTL_SAVE_CHARACTOR")
        const lines = text.split("\n")
        let count = 0
        // 最後の要素は「最後の改行より後ろ」なので、まだ確定していない行として数えない
        for (let i = 0; i < lines.length - 1; i++) {
            if (lines[i] === ja_marker || lines[i] === KFTL_ASCII_SAVE_CHARACTOR) {
                count++
            }
        }
        return count
    }

    // 全角「！」または半角「!」の保存マーカーを取り除く
    function remove_save_marker(text: string): string {
        const ja_marker = "\n" + i18n.global.t("KFTL_SAVE_CHARACTOR") + "\n"
        if (text.includes(ja_marker)) {
            return text.replace(ja_marker, "\n")
        }
        return text.replace("\n" + KFTL_ASCII_SAVE_CHARACTOR + "\n", "\n")
    }

    /** textarea の入力イベント。ここで印をつけ、watch 側で保存マーカーを判定する */
    function onTextAreaInput(): void {
        user_input_tab_id = active_tab_id.value
    }

    /**
     * 保存マーカーの行が**増えていたら**送信する。
     *
     * 入口は「textarea の `@input` 起点の watch」と「テンプレート貼り付け」の2つだけ。
     * 判定そのものはここ1箇所に閉じている。
     *
     * **「末尾がマーカーか」で見てはいけない。** watch は flush:'post' なので、
     * 1回のフラッシュ窓の中で本文が2回変わると **1回しか呼ばれず、中間の値
     * (マーカーで終わっている本文)は一度も観測されない**。行数の多いタブでは
     * 解析(`get_invalid_line_indexs` は行ごとに await)がメインスレッドを掴むので、
     * その間に打たれたキーがまとめて着地して現実に起きる ―― これが
     * 「素早く入力すると \n！\n が反応しない」の正体。
     * 末尾で見ていると、その窓では既に末尾がマーカーではないので黙って落ちる。
     *
     * 代わりに「確定したマーカー行の数が前より増えたか」で見る。こうすると
     *   - 1回のフラッシュ窓でマーカーの後ろまで打たれても拾える
     *   - マーカーが1行目にあっても拾える(前後の改行を要求しないので)
     *   - 既にマーカーが残っている本文を編集しただけでは増えないので再送信しない
     * が同時に成り立つ。
     *
     * `previous_content` は比較の基準。watch からは old_value を、
     * テンプレート貼り付けからは貼る前の本文を渡す。
     */
    async function maybe_submit_by_save_marker(content: string, previous_content: string): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        if (count_save_marker_lines(content) <= count_save_marker_lines(previous_content)) {
            return
        }
        await submit()
    }

    // 追加されるタグのうち、TagStructに存在しないものを重複なく集める。
    // 実在判定は共有ゲート(use-confirm-unknown-tag.ts)に任せ、ここは行から集めるだけ
    function collect_unknown_tags(kftl_requests: Array<KFTLRequest>): Array<string> {
        const candidates = new Array<string>()
        for (let i = 0; i < kftl_requests.length; i++) {
            candidates.push(...kftl_requests[i].get_tags())
        }
        return confirm_unknown_tag.collect_unknown_tags(candidates)
    }

    async function submit(): Promise<void> {
        // 新しい送信は必ずアクティブなタブが対象。持ち越した値は見ない
        await do_submit(active_tab_id.value, false, false)
    }

    function cancel_submit(): void {
        confirm_unknown_tag.close_confirm()
        is_confirm_unknown_tag_open.value = false
        submit_target_tab_id.value = null
    }

    async function confirm_submit(): Promise<void> {
        const target_tab_id = submit_target_tab_id.value ?? active_tab_id.value
        confirm_unknown_tag.close_confirm()
        is_confirm_unknown_tag_open.value = false
        submit_target_tab_id.value = null
        // タグの確認を通しただけ。板名の確認はこの後の do_submit で改めて出る
        await do_submit(target_tab_id, true, false)
    }

    /**
     * 確認ダイアログが閉じた。保存・キャンセル・×・Escape・ブラウザバックのどれでも来る。
     *
     * `cancel_submit` / `confirm_submit` も自分でロックを倒すので、これは
     * **ブラウザバックのような明示の経路を通らない閉じ方**のための保険。
     * これが無いと `unknown_tags` の空判定と同じで、タブが二度と切り替えられなくなる。
     *
     * **`submit_target_tab_id` はここで消してはいけない。**
     * ダイアログの「保存」は `hide()` してから `requested_confirm` を出すので、
     * ここが先に走る。消すと `confirm_submit()` が持ち越した対象を見失い、
     * 確認中に切り替えたタブへ誤配送される。
     * ブラウザバックで古い値が残っても、新しい送信は必ずアクティブなタブを渡すので害はない
     */
    function onConfirmUnknownTagClosed(): void {
        is_confirm_unknown_tag_open.value = false
    }

    function cancel_mi_board_submit(): void {
        confirm_unknown_mi_board.close_confirm()
        submit_target_tab_id.value = null
    }

    async function confirm_mi_board_submit(): Promise<void> {
        const target_tab_id = submit_target_tab_id.value ?? active_tab_id.value
        confirm_unknown_mi_board.remember_confirmed_mi_boards()
        confirm_unknown_mi_board.close_confirm()
        submit_target_tab_id.value = null
        await do_submit(target_tab_id, true, true)
    }

    // 保存本体。KFTLは複数リクエストをtxで束ねて送るので、二重送信すると
    // Kyouが丸ごと重複登録される。フラグはここで立てる
    // （テンプレートの :disabled / :readonly はこのフラグを見ている。
    //   以前は保存マーカー検出経路でしか立てておらず、保存ボタン経由では実質ノーガードだった）
    async function do_submit(target_tab_id: string, skip_unknown_tag_check: boolean, skip_unknown_mi_board_check: boolean): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        // 同じタブを映しているもう1枚のウィンドウが送信中なら見送る。
        // フラグはビューごとなので、ウィンドウをまたいだ排他は共有ストアにしか置けない。
        //
        // 掴む位置を間違えると壊れる:
        //   - `is_requested_submit` のガードより前に掴むと、上の return が finally を通らず
        //     掴んだままになり、そのタブが全ウィンドウで永久に保存できなくなる
        //   - try の中で掴むと、掴めなかった側も finally を通るので
        //     **勝ったウィンドウの分を解放**してしまい、排他が無意味になる
        if (!tabs_store.try_begin_submit(target_tab_id)) {
            return
        }
        const get_submitting_content = () => tabs_store.get_tab_content(target_tab_id)
        const set_submitting_content = (content: string) => tabs_store.set_tab_content(target_tab_id, content)

        is_requested_submit.value = true
        is_submitting.value = true
        try {
            // 表示用の invalid_line_numbers はアクティブなタブのもので、しかも await をまたいで
            // 遅れて着地する。送信の可否は送信対象タブから引き直して判定する
            const invalid_lines = await new KFTLStatement(get_submitting_content()).get_invalid_line_indexs()
            if (invalid_lines.length != 0) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.kftl_has_invalid_line
                error.error_message = i18n.global.t("KFTL_FOUND_INVALID_LINE_MESSAGE")
                set_submitting_content(remove_save_marker(get_submitting_content()))
                emits('received_errors', [error])
                return
            }
            const statement = new KFTLStatement(get_submitting_content())
            const kftl_requests = await statement.generate_requests()

            // TagStructに存在しないタグを検出したら、送信前に確認を取る
            if (!skip_unknown_tag_check) {
                const not_found = collect_unknown_tags(kftl_requests)
                if (not_found.length > 0) {
                    // 保存マーカーを消しておかないと、確認中の入力で再度submitされてしまう
                    set_submitting_content(remove_save_marker(get_submitting_content()))
                    submit_target_tab_id.value = target_tab_id
                    is_confirm_unknown_tag_open.value = true
                    confirm_unknown_tag.open_confirm(not_found)
                    return
                }
            }

            // まだ実在しない板名を検出したら、送信前に確認を取る。
            // 板名行は自由入力なので、打ち間違いがそのまま新しい板になってしまう。
            // タグの確認を通した後に改めてここへ来る（確認は1つずつ順に出す）
            if (!skip_unknown_mi_board_check) {
                const board_names = kftl_requests.map(kftl_request => kftl_request.get_mi_board_name())
                const not_found_boards = confirm_unknown_mi_board.collect_unknown_mi_boards(board_names)
                if (not_found_boards.length > 0) {
                    // タグ確認と同じ理由で保存マーカーを消しておく
                    set_submitting_content(remove_save_marker(get_submitting_content()))
                    submit_target_tab_id.value = target_tab_id
                    confirm_unknown_mi_board.open_confirm(not_found_boards)
                    return
                }
            }

            let last_added_request_time = new Date(Date.now()) // 「、、」でずれた分をPlaingTimeIsにわたすための考慮。リロード時刻より大きかった場合はこの値でTimeIsをリロードする
            let errors = new Array<GkillError>()
            const result_kyou_ids = new Array<KFTLRequestResult>()
            const tx_id = kftl_requests.length > 0 ? kftl_requests[0].get_tx_id() : null
            for (let i = 0; i < kftl_requests.length; i++) {
                const request = kftl_requests[i]
                const request_related_time = request.get_related_time()
                if (request_related_time && request_related_time.getTime() > last_added_request_time.getTime()) {
                    last_added_request_time = request_related_time
                }
                await request.do_request(props.gkill_api, props.application_config).then(request_errors => errors = errors.concat(request_errors))
                result_kyou_ids.push(...request.get_result_kyou_ids())
            }
            if (errors.length != 0) {
                emits('received_errors', errors)
                set_submitting_content(remove_save_marker(get_submitting_content()))

                if (tx_id) {
                    const deiscard_req = new DiscardTXRequest()
                    deiscard_req.tx_id = tx_id
                    const discard_res = await props.gkill_api.discard_tx(deiscard_req)
                    if (discard_res.errors && discard_res.errors.length != 0) {
                        emits('received_errors', discard_res.errors)
                    }
                    return
                }
            }
            if (tx_id) {
                const commit_req = new CommitTXRequest()
                commit_req.tx_id = tx_id
                const commit_res = await props.gkill_api.commit_tx(commit_req)
                if (commit_res.errors && commit_res.errors.length != 0) {
                    emits('received_errors', commit_res.errors)
                    set_submitting_content(remove_save_marker(get_submitting_content()))
                    return
                }
            }

            // 保存できたタブは閉じる。0枚になるなら空のタブが1枚できる（use-kftl-tabs.ts）
            tabs_store.close_tab(target_tab_id)
            focus_active_tab_editor()
            const message = new GkillMessage()
            message.message_code = GkillMessageCodes.saved_kftls
            message.message = i18n.global.t("SAVED_MESSAGE")
            emits('received_messages', [message])
            emits('saved_kyou_by_kftl', last_added_request_time)
            // 保存はcommitで完了している。この先は一覧へ知らせるための引き直しだけなので、
            // 入力欄と各ボタンをreadonly/disabledのまま待たせない
            // （引き直しはKyouの件数ぶん往復するので、待たせると体感で数秒固まる）
            is_requested_submit.value = false
            is_submitting.value = false
            await emit_saved_kyous(result_kyou_ids)
        } finally {
            // 解放点はここ1箇所だけ。タグ確認・板名確認で抜ける return もここを通るので
            // 確認待ちの間は手放され、confirm_submit / confirm_mi_board_submit からの
            // 再入で取り直す（**持ち越すと再入で自己デッドロックする**）。
            // 確認が開いている隙に別ウィンドウが同じタブを送れるが、そのときは本文から
            // 保存マーカーが除去済みで、後続の確認は消えたタブを対象にするため
            // get_tab_content が "" を返してリクエスト0件で無害に終わる
            tabs_store.end_submit(target_tab_id)
            is_requested_submit.value = false
            is_submitting.value = false
        }
    }

    /**
     * commit後に、作った / 更新した Kyou を引き直して上げる。
     *
     * tx中は add_* が added_kyou を返せない（一時リポジトリにしか無い）ので、
     * commitを終えたここで初めて実体が手に入る。
     * commitより前に引くと「まだ無い」応答をServiceWorkerのPOSTキャッシュが
     * 掴んでしまうため、引く前にそのidのキャッシュを捨てる。
     *
     * 他のAdd系ダイアログと同じく registered_kyou で上げるので、
     * rykv / mi / dashboard 側は KFTL 専用の分岐を持たなくてよい。
     */
    async function emit_saved_kyous(results: ReadonlyArray<KFTLRequestResult>): Promise<void> {
        if (results.length === 0) {
            return
        }
        const kyous = await Promise.all(results.map(async (result) => {
            try {
                await delete_gkill_kyou_cache(result.id)
            } catch (_err: unknown) {
                // Cache API が使えない環境ではスキップ
            }
            const req = new GetKyouRequest()
            req.id = result.id
            const res = await props.gkill_api.get_kyou(req)
            if (res.errors && res.errors.length !== 0) {
                return null
            }
            return res.kyou_histories[0] ?? null
        }))

        let is_emitted = false
        for (let i = 0; i < results.length; i++) {
            const kyou = kyous[i]
            if (!kyou) {
                continue
            }
            is_emitted = true
            if (results[i].kind === 'updated') {
                emits('updated_kyou', kyou)
            } else {
                emits('registered_kyou', kyou)
            }
        }
        // 1件も引けなかったときだけ、従来どおりリスト全体の引き直しへ落とす
        if (!is_emitted) {
            emits('requested_reload_list')
        }
    }

    function show_kftl_template_dialog(): void {
        kftl_template_dialog.value?.show()
    }

    /**
     * テンプレートは上書きではなく新しいタブで開く。
     * 表示名は kftl-template-view.vue のボタンに出ている `title` を使う
     */
    function paste_template(template: KFTLTemplateElementData): Promise<void> {
        capture_editor_view_state()
        const template_name = template.title !== "" ? template.title
            : (template.name !== "" ? template.name : null)
        // 新しいタブなので、比較の基準は「マーカーが1つも無い状態」。
        // テンプレートがマーカー行を持っていれば増えたことになる
        const content_before_paste = ""
        active_tab_id.value = tabs_store.add_tab(template.template as string, template_name)
        kftl_template_dialog.value?.hide()
        focus_active_tab_editor()
        // テンプレート選択は利用者の明示的な操作なので、保存マーカーで終わっていればそのまま送信する。
        //
        // `user_input_tab_id` を立てて watch に任せてはいけない ―― watch は
        // `new_value === old_value` で早期returnするので、貼る前のタブの本文が
        // テンプレートと同一文字列だと黙って発火しない（タブ化する前も同じ理由で取りこぼしていた）。
        // また watch は flush: 'post' かつ await を挟むので、判定までにタブを切り替えられると
        // 「印のタブ == アクティブタブ」が偽になって、これも黙って発火しない。
        // ここは add_tab / active_tab_id 代入が同期で済んでいるので、直接呼べば窓が開かない
        return maybe_submit_by_save_marker(text_area_content.value, content_before_paste)
    }

    async function focus_kftl_text_area(): Promise<void> {
        kftl_text_area.value?.focus()
    }

    // ── Event relay objects ──
    const errorMessageRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    // ── Return ──
    return {
        // Template refs
        kftl_template_dialog,
        confirm_close_kftl_tab_dialog,
        kftl_text_area,
        kftl_line_label_wrap,
        confirm_unknown_mi_board_dialog: confirm_unknown_mi_board.confirm_unknown_mi_board_dialog,
        confirm_unknown_tag_dialog: confirm_unknown_tag.confirm_unknown_tag_dialog,

        // Confirm unknown mi board
        unknown_mi_boards: confirm_unknown_mi_board.unknown_mi_boards,
        cancel_mi_board_submit,
        confirm_mi_board_submit,

        // Tabs
        tabs,
        active_tab_id,
        active_tab_id_model,
        is_tab_locked,
        tab_bar_height,
        tab_label,
        add_tab,
        activate_tab,
        request_close_tab,
        confirm_close_tab,
        cancel_close_tab,
        pending_close_tab_label,

        // State
        text_area_content,
        text_area_element_id,
        line_label_datas,
        line_label_styles,
        is_requested_submit,
        title_height,
        unknown_tags: confirm_unknown_tag.unknown_tags,
        is_confirm_unknown_tag_open,

        // Computed
        text_area_width_px,
        text_area_height_px,
        line_label_width_px,
        line_label_height_px,
        kftl_input_height_px,
        kftl_input_width_px,
        title_height_px,

        // Business logic
        submit,
        cancel_submit,
        confirm_submit,
        onConfirmUnknownTagClosed,
        show_kftl_template_dialog,
        paste_template,
        focus_kftl_text_area,
        onTextAreaInput,
        update_line_labels,

        // Event relay objects
        errorMessageRelayHandlers,
    }
}
