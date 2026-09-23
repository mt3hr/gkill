/**
 * ドラッグ＆ドロップの並べ替えで「どこに入るか」を見せる線（挿入位置の表示）。
 *
 * 構造ツリー（タグ・記録保管場所・プロファイル・記録タイプ・板・テンプレート）と
 * 集計ビュー・関連情報の編集画面の並べ替えで共有する。以前はドラッグ中の見た目の変化が
 * ブラウザ標準の半透明の像しか無く、どこに入るのか分からなかった（利用者報告）。
 *
 * 守ること:
 * - **線の位置と実際の挿入位置は同じ関数（decide_drop_position）で決める。** dragover で線を出し、
 *   drop でも同じ関数で入れる先を決める。別々に書くと「線は下なのに上に入る」が起きる
 *   （構造ツリーは drop だけが offsetY と固定の行の高さ 24px で判定していて、境界がずれていた）
 * - 状態は「いま線を出している要素1つ」だけをモジュールに持ち、クラスを直接付け外しする。
 *   構造ツリーは数百ノードを再帰で描くので、リアクティブな状態を全ノードへ配らない
 * - ドロップ先がこの仕組みの外でも、ドラッグを取り消しても（Escape・ウィンドウの外）線を残さない。
 *   window の capture で drop / dragend を受けて消す
 */

/** 入る位置。before=対象の直前、inside=フォルダの中（末尾）、after=対象の直後 */
export type DropPosition = 'before' | 'inside' | 'after'

export const DROP_INDICATOR_CLASS: Readonly<Record<DropPosition, string>> = {
    before: 'gkill-drop-before',
    inside: 'gkill-drop-inside',
    after: 'gkill-drop-after',
}

/** 掴んでいる元の要素に付けるクラス（半透明にする） */
export const DRAG_SOURCE_CLASS = 'gkill-drag-source'

/**
 * 対象の矩形とポインタの縦位置から入る位置を決める。
 * 中へ入れられる対象（フォルダ）は上1/3=直前・中1/3=中・下1/3=直後、そうでなければ上半分=直前・下半分=直後。
 * 境界は「以下」を上側に含める（以前の判定と同じ）。矩形の外は近いほうへ寄せる。
 */
export function decide_drop_position(rect: { top: number, height: number }, client_y: number, accepts_inside: boolean): DropPosition {
    const y = client_y - rect.top
    if (accepts_inside) {
        if (y <= rect.height / 3) {
            return 'before'
        }
        if (y <= rect.height * 2 / 3) {
            return 'inside'
        }
        return 'after'
    }
    return y <= rect.height / 2 ? 'before' : 'after'
}

let indicator_element: HTMLElement | null = null
let indicator_position: DropPosition | null = null
let drag_source_element: HTMLElement | null = null
let is_cleanup_installed = false

function install_cleanup(): void {
    if (is_cleanup_installed || typeof window === 'undefined') {
        return
    }
    is_cleanup_installed = true
    window.addEventListener('drop', () => end_drag(), true)
    window.addEventListener('dragend', () => end_drag(), true)
}

/** el に線を出す。前に出していた線は消す（線は常に1本） */
export function show_drop_indicator(el: HTMLElement, position: DropPosition): void {
    install_cleanup()
    if (indicator_element === el && indicator_position === position) {
        return
    }
    hide_drop_indicator()
    el.classList.add(DROP_INDICATOR_CLASS[position])
    indicator_element = el
    indicator_position = position
}

/**
 * 線を消す。el を渡したときは、その要素に出している線だけを消す
 * （子の行へ移ったあとで親の dragleave が届いても、子の線を消さないため）
 */
export function hide_drop_indicator(el?: HTMLElement | null): void {
    if (indicator_element === null) {
        return
    }
    if (el && el !== indicator_element) {
        return
    }
    indicator_element.classList.remove(DROP_INDICATOR_CLASS.before, DROP_INDICATOR_CLASS.inside, DROP_INDICATOR_CLASS.after)
    indicator_element = null
    indicator_position = null
}

/** ドラッグの開始。掴んだ元を覚えて半透明にする */
export function begin_drag_source(el: HTMLElement | null): void {
    install_cleanup()
    end_drag()
    if (el === null) {
        return
    }
    drag_source_element = el
    // ドラッグ中の像は dragstart の時点の見た目から作られる。すぐに半透明にすると像まで薄くなるので次のタスクで付ける
    setTimeout(() => {
        if (drag_source_element === el) {
            el.classList.add(DRAG_SOURCE_CLASS)
        }
    }, 0)
}

/** ドラッグの終わり（ドロップ・取り消し）。線と掴んでいる元の表示を両方消す */
export function end_drag(): void {
    hide_drop_indicator()
    if (drag_source_element !== null) {
        drag_source_element.classList.remove(DRAG_SOURCE_CLASS)
        drag_source_element = null
    }
}

/**
 * el が掴んでいる元そのもの、またはその中にあるか。
 * 自分自身や、フォルダを自分の子孫へ落とす位置には入れられないので線を出さない
 */
export function is_inside_drag_source(el: HTMLElement): boolean {
    return drag_source_element !== null && drag_source_element.contains(el)
}

/** dragleave が要素の外へ出たものか（子の要素へ移っただけなら偽） */
export function is_leaving_element(e: DragEvent): boolean {
    const current = e.currentTarget
    const related = e.relatedTarget
    if (!(current instanceof Node) || !(related instanceof Node)) {
        return true
    }
    return !current.contains(related)
}

/** ドラッグしている物がこの並べ替えのものか（dragover では中身を読めないので種類だけ見る） */
export function has_drag_type(e: DragEvent, type: string): boolean {
    const types = e.dataTransfer?.types
    if (!types) {
        return false
    }
    return Array.from(types).includes(type)
}
