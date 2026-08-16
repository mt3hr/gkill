'use strict'

import type { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'
import { FOLDABLE_STRUCT_ROOT_KEY, is_struct_container_node } from '@/pages/views/foldable-struct-model'

/**
 * 「すべて」板の番兵キー。
 *
 * ApplicationConfig の板ツリーに append_all_mi_board() が入れるノードの key/board_name が
 * この値で、FoldableStruct が emit するのも key なので、番兵は**ロケール非依存**でなければ
 * ならない。以前は i18n の MI_ALL_BOARD_NAME_TITLE と比較していたため、日本語以外の
 * ロケールでは「すべて」をクリックしても全件(mi_board_name=null)に戻らず、
 * 「すべて」という名前の板で絞り込まれて0件になっていた。
 * 表示名は i18n(ALL_MI_BOARD_NAME / MI_ALL_TITLE)のままでよい。
 */
export const MI_ALL_BOARD_KEY = "すべて"

/**
 * 板ツリーを DFS して board_name を設定順に並べる。
 *
 * 並び順は children 配列の順そのもの(設定ダイアログの「上へ / 下へ」の順)。
 * board_name が空のノード(ルート等)は板ではないので含めない。
 */
export function collect_mi_board_names_in_config_order(struct: MiBoardStructElementData | null): Array<string> {
    const board_names = new Array<string>()
    if (!struct) {
        return board_names
    }
    const walk = (node: MiBoardStructElementData): void => {
        if (node.board_name !== "") {
            board_names.push(node.board_name)
        }
        node.children?.forEach(child => {
            if (child) {
                walk(child)
            }
        })
    }
    walk(struct)
    return board_names
}

/**
 * 板ツリーのクリックで上がってきた key 列から、実際に開く板名を決める。
 *
 * FoldableStruct の clicked_items は
 *  - リーフのクリック  → そのノードの key 1つだけ（click_item_by_user）
 *  - グループ行のクリック → **自分自身の key を先頭に**、配下のノードの key 全部（click_group_by_user）
 * を載せてくる。板ツリーはルートを folder_name="" のクリック可能な帯として描いているので、
 * その空白を踏むと `__root__` と全部の板が一度に飛んできて、
 * `__root__` という名前の列 + 板の数だけの列が開いていた。
 *
 * 判定は「グループ行のクリックなら何も開かない」。グループ行かどうかは、
 * ツリー上でフォルダ扱いのノード（is_dir、または board_name を持たないルート）の key が
 * 混ざっているかで見る。板ツリーはいま平坦なのでフォルダはルートだけだが、
 * フォルダを持てるようになってもフォルダ名の列を開かない。
 *
 * ツリーに無い key は**開く**。板を作った直後で append_not_found_mi_boards() が
 * まだ拾えていないだけかもしれず、ここで落とすと「板をクリックしても何も起きない」
 * （エラーも出ない）になる。
 */
export function resolve_clicked_mi_board_names(
    items: Array<string>,
    struct: MiBoardStructElementData | null,
): Array<string> {
    // ルートキーはツリーを見なくても板ではないと分かる。
    // 設定の読み込み前(struct=null)や、保存済みJSONのルートに is_dir が付いていない
    // 場合（そのときルートはリーフとして描かれ、name/key の __root__ がそのまま出る）でも
    // 開かせないために、ここは struct と独立に弾く
    const not_board_keys = new Set<string>([FOLDABLE_STRUCT_ROOT_KEY])
    const walk = (node: MiBoardStructElementData): void => {
        if (is_struct_container_node(node) || node.board_name === "") {
            not_board_keys.add(node.key)
        }
        node.children?.forEach(child => {
            if (child) {
                walk(child)
            }
        })
    }
    if (struct) {
        walk(struct)
    }

    const board_names = new Array<string>()
    for (const item of items) {
        if (item === "" || not_board_keys.has(item)) {
            // グループ行のクリック。配下の板をまとめて開いたりはしない
            return []
        }
        if (!board_names.includes(item)) {
            board_names.push(item)
        }
    }
    return board_names
}

/**
 * サーバから来た板名一覧を ApplicationConfig の設定順に並べ替える。
 *
 * get_mi_board_list はマップ反復順で返す(Go 側の doc コメントにも「並び順は保証しません」と
 * 明記されている)ので、素で :items に渡すとプルダウンの並びが呼ぶたびに変わる。
 * 順序を持っているのは ApplicationConfig の板ツリーだけなので、ここで並べ直す。
 *
 * - 設定にある板が先。設定の children 順
 * - 設定に無い板(設定の読み込み後に作られた板など)は元の順のまま末尾へ
 * - **設定にしか無い名前は足さない**。とくに「すべて」は実在の板ではないので、
 *   実際にその名前の板がサーバから返ってきたときだけ残る
 * - 重複は除く
 * - struct が未設定(設定の読み込み前)でも落ちない。そのときは入力の順のまま返す
 */
export function sort_mi_board_names_by_config_order(
    board_names: Array<string>,
    struct: MiBoardStructElementData | null,
): Array<string> {
    const remaining = new Set(board_names)
    const sorted = new Array<string>()

    collect_mi_board_names_in_config_order(struct).forEach(board_name => {
        if (remaining.delete(board_name)) {
            sorted.push(board_name)
        }
    })
    board_names.forEach(board_name => {
        if (remaining.delete(board_name)) {
            sorted.push(board_name)
        }
    })
    return sorted
}
