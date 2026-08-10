'use strict'

import type { InfoIdentifier } from "@/classes/datas/info-identifier"
import type { GkillPropsBase } from "../views/gkill-props-base"

export interface KyouListViewDialogProps extends GkillPropsBase {
    highlight_targets: Array<InfoIdentifier>
    list_height: number
    enable_context_menu: boolean
    enable_dialog: boolean
    force_show_latest_kyou_info: boolean
    show_rep_name: boolean
    /**
     * このダイアログが自分でrykvダイアログをホストするか（既定: true）。
     *
     * 自分でホストすると、中で起きたタグ追加等の requested_reload_kyou が
     * このダイアログのサブツリー内で完結するので、抱えているリストに反映できる。
     * false にすると従来どおり requested_open_rykv_dialog を親へ中継する
     * （v-virtual-scroll の行使い回しでこのダイアログごとunmountされる場合の逃げ道）。
     */
    host_rykv_dialogs?: boolean
}
