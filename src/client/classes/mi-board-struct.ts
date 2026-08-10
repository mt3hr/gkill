'use strict'

import { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'
import { GkillAPI } from '@/classes/api/gkill-api'

// MiBoardStruct（板構成）のツリーに指定された板名が存在するかを再帰的に調べる
export function board_exists_in_mi_board_struct(board_name: string, struct: MiBoardStructElementData): boolean {
    if (struct.board_name === board_name) return true
    if (struct.children) {
        for (const child of struct.children) {
            if (board_exists_in_mi_board_struct(board_name, child)) return true
        }
    }
    return false
}

// 板名をツリーの直下へ足す。既にあれば何もしない。
// ApplicationConfig.append_not_found_mi_boards() が読み込み時にやっているのと同じことを、
// 「新しい板です」の確認を通した直後にその場でやる。
// これをしないと、同じ新しい板へ2件目を入れるときにまた確認が出る。
// サーバへは保存しない（次回の読み込みで append_not_found_mi_boards が API から拾い直す）
export function append_mi_board_to_struct(board_name: string, struct: MiBoardStructElementData): void {
    if (board_name === "" || board_exists_in_mi_board_struct(board_name, struct)) {
        return
    }
    const board_struct = new MiBoardStructElementData()
    board_struct.key = board_name
    board_struct.name = board_name
    board_struct.check_when_inited = true
    board_struct.is_checked = board_struct.check_when_inited
    board_struct.id = GkillAPI.get_gkill_api().generate_uuid()
    board_struct.board_name = board_name
    if (!struct.children) {
        struct.children = []
    }
    struct.children.push(board_struct)
}
