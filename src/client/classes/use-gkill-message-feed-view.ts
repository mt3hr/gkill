import { computed } from 'vue'
import { i18n } from '@/i18n'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { useGkillMessageFeed, type GkillFeedItem, type GkillFeedLevel } from '@/classes/use-gkill-message-feed'

// 画面右上のエラー / メッセージ一覧（gkill-message-feed-view.vue）のロジック。
// 一覧そのものは use-gkill-message-feed.ts のシングルトンで、ここは表示とコピーだけ。
export function useGkillMessageFeedView() {
    const { feed_items, push_messages, close_feed_item, format_feed_item_for_copy } = useGkillMessageFeed()

    const has_items = computed(() => feed_items.value.length !== 0)

    // v-alert の色。info は色を付けない（従来どおり）
    function alert_color(level: GkillFeedLevel): string | undefined {
        switch (level) {
            case 'error':
                return 'error'
            case 'warning':
                return 'warning'
            default:
                return undefined
        }
    }

    // エラーだけ role="alert"（支援技術へ即時通知。E2E もこの形で掴んでいる）
    function alert_role(level: GkillFeedLevel): string | undefined {
        return level === 'error' ? 'alert' : undefined
    }

    function show_footer(item: GkillFeedItem): boolean {
        return item.level !== 'info'
    }

    // ── Event handlers ──
    function onClickClose(id: string): void {
        close_feed_item(id)
    }

    async function onClickCopy(item: GkillFeedItem): Promise<void> {
        const text = format_feed_item_for_copy(item, typeof window !== 'undefined' ? window.location.pathname : '')
        try {
            await navigator.clipboard.writeText(text)
        } catch (_e) {
            // クリップボードが使えない環境（http の LAN アクセス等）。何もしないより控えられる形で出す
            window.prompt(i18n.global.t('ERROR_ALERT_COPY_TITLE'), text)
            return
        }
        const copied = new GkillMessage()
        copied.message_code = GkillMessageCodes.copied_error_detail
        copied.message = i18n.global.t('ERROR_ALERT_COPIED_MESSAGE')
        push_messages([copied])
    }

    return {
        // State
        feed_items,
        has_items,

        // Template helpers
        alert_color,
        alert_role,
        show_footer,

        // Event handlers
        onClickClose,
        onClickCopy,
    }
}
