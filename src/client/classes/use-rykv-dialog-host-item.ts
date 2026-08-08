import { computed, nextTick, onMounted, ref } from 'vue'
import type { Notification } from '@/classes/datas/notification'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Kyou } from '@/classes/datas/kyou'
import type { OpenedRykvDialog } from '@/pages/views/rykv-dialog-kind'
import type { GkillPropsBase } from '@/pages/views/gkill-props-base'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import type { ComponentRef } from '@/classes/component-ref'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'

interface RykvDialogHostItemProps extends GkillPropsBase {
    item: OpenedRykvDialog
    enable_context_menu: boolean
    enable_dialog: boolean
}

interface RykvDialogHostItemEmits extends KyouDialogEmits {
    (e: 'closed', id: string): void
}

export function useRykvDialogHostItem(options: {
    props: RykvDialogHostItemProps,
    emits: RykvDialogHostItemEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const dialog = ref<ComponentRef | null>(null)

    // ── Computed ──
    const payload_tag = computed(() => (props.item.payload ?? null) as Tag | null)
    const payload_text = computed(() => (props.item.payload ?? null) as Text | null)
    const payload_notification = computed(() => (props.item.payload ?? null) as Notification | null)

    // ── Event relay objects ──
    const dialog_events = {
        ...build_kyou_dialog_relay(emits, {
            // クリックはフォーカス移動も伴う
            'clicked_kyou': (kyou: Kyou) => {
                emits('focused_kyou', kyou)
                emits('clicked_kyou', kyou)
            },
        }),
        // closed はダイアログ固有なので共通束には含まれない
        closed: () => emits('closed', props.item.id),
    }

    // ── Lifecycle ──
    onMounted(async () => {
        await nextTick()
        // payload必須のkindをpayloadなしで開くと、rykv-dialog-host-item.vue の
        // v-else-ifチェーンがどれも当たらず何も描画されない。そのまま放置すると
        // closed が永久に飛ばず、opened_dialogs に不可視のエントリが残り続ける。
        // kindごとの必須payloadを列挙するより「描画されなかった事実」を見るほうが、
        // 将来kindが増えても取りこぼさない
        if (!dialog.value) {
            emits('closed', props.item.id)
            return
        }
        dialog.value.show?.()
    })

    // ── Return ──
    return {
        // Template refs
        dialog,

        // Computed
        payload_tag,
        payload_text,
        payload_notification,

        // Event relay objects
        dialog_events,
    }
}
