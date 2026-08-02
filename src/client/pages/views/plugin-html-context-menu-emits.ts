'use strict'

import type { KyouViewEmits } from "./kyou-view-emits"

export interface PluginHtmlContextMenuEmits extends KyouViewEmits {
    // プラグインの設定ダイアログを開く。rep_name でどのプラグインの設定かが決まる。
    (e: 'requested_show_plugin_config', rep_name: string): void
}
