'use strict'

import type { KyouViewEmits } from "./kyou-view-emits"

// プラグイン本文のコンテキストメニューは KyouView と同じイベントしか出さない。
// プラグイン設定を開く固有イベントは、設定の入口がアカウント管理画面へ移ったため撤去した（ADR-0714）
export type PluginHtmlContextMenuEmits = KyouViewEmits
