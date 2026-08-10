'use strict'

/**
 * ダイアログを開いたときに最初のテキスト入力欄へフォーカスを当てるための探索。
 *
 * 入力欄の大半は `pages/views/*.vue` 側にあり、その view はサイドバーやページ直下でも
 * 使われる。view に `autofocus` を撒くとダイアログ以外でもページ読込時にフォーカスを
 * 奪ってしまうので、判定はダイアログの共通基盤 (`use-floating-dialog.ts`) 側に置く。
 *
 * 探索は `.gkill-floating-dialog__body` の中に限定する。ヘッダには透過トグルの
 * v-checkbox と×ボタンが必ず先頭にあり、ルートから素直に探すと必ずそれを掴む。
 */

/** ダイアログ本文。ここより外（ヘッダ）は探索しない */
export const DIALOG_BODY_SELECTOR = '.gkill-floating-dialog__body'

/**
 * 文字を打ち込める input の type。
 * v-select の内部 input は readonly なので type では弾かず readonly で弾く。
 */
const TEXT_INPUT_TYPES = new Set([
    '',
    'text',
    'search',
    'url',
    'email',
    'tel',
    'password',
    'number',
    'date',
    'time',
    'datetime-local',
    'month',
    'week',
])

/**
 * 要素が実際に見えているか。
 *
 * `v-show` は `display: none` をインラインで当てるので、祖先まで遡って調べる。
 * `offsetParent === null` で判定するほうが手軽だが、jsdom はレイアウトしないため
 * 常に null になり単体テストが書けない。計算スタイルなら両方で同じ結果になる。
 */
function is_visible(element: HTMLElement): boolean {
    if (element.hidden) {
        return false
    }
    for (let node: HTMLElement | null = element; node; node = node.parentElement) {
        const style = window.getComputedStyle(node)
        if (style.display === 'none' || style.visibility === 'hidden') {
            return false
        }
    }
    return true
}

function is_text_entry(element: HTMLElement): boolean {
    if (element instanceof HTMLTextAreaElement) {
        return true
    }
    if (!(element instanceof HTMLInputElement)) {
        return false
    }
    // v-checkbox / v-switch / v-radio の内部 input を掴まない
    if (element.closest('.v-selection-control')) {
        return false
    }
    // v-select は input を持つが文字は打ち込めない（Vuetify が inputmode="none" を付ける）。
    // v-autocomplete / v-combobox は打ち込めるので inputmode は付かず、ここは通る
    if (element.getAttribute('inputmode') === 'none') {
        return false
    }
    return TEXT_INPUT_TYPES.has(element.type.toLowerCase())
}

function is_enabled(element: HTMLInputElement | HTMLTextAreaElement): boolean {
    // readonly は「日付ピッカーを開くだけの見せかけの入力欄」でも使われている。
    // ここにフォーカスするとダイアログを開いた瞬間にピッカーが開いてしまう
    return !element.disabled && !element.readOnly
}

/**
 * ダイアログを開いたときにフォーカスすべき要素を返す。無ければ null。
 *
 * - 既に `autofocus` を付けた要素があるダイアログでは null を返す
 *   （Vuetify の autofocus に任せる。二重にフォーカスしない）
 * - 探索は `.gkill-floating-dialog__body` の中だけ。root 自身が本文でもよい
 */
export function find_autofocus_target(root: HTMLElement | null): HTMLElement | null {
    if (!root) {
        return null
    }
    const body = root.matches(DIALOG_BODY_SELECTOR) ? root : root.querySelector(DIALOG_BODY_SELECTOR)
    if (!(body instanceof HTMLElement)) {
        return null
    }
    if (body.querySelector('[autofocus]')) {
        return null
    }
    for (const candidate of body.querySelectorAll<HTMLElement>('input, textarea')) {
        if (!is_text_entry(candidate)) {
            continue
        }
        if (!is_enabled(candidate as HTMLInputElement | HTMLTextAreaElement)) {
            continue
        }
        if (!is_visible(candidate)) {
            continue
        }
        return candidate
    }
    return null
}

/**
 * 既にダイアログ内の入力欄にフォーカスが入っているか。
 * 入力欄が遅れて生えるダイアログでは後追いでフォーカスするので、
 * その間にユーザーが自分で入力欄を選んでいたら手を出さない。
 */
export function has_focus_inside(root: HTMLElement | null): boolean {
    if (!root) {
        return false
    }
    const active = document.activeElement
    if (!(active instanceof HTMLElement) || active === document.body) {
        return false
    }
    return root.contains(active) && is_text_entry(active)
}
