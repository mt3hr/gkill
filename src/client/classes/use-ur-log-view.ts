import { computed, ref } from 'vue'
import noimage from '@/assets/noimage.webp'
import type { URLogViewProps } from '@/pages/views/ur-log-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { ComponentRef } from '@/classes/component-ref'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useURLogView(options: {
    props: URLogViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref<ComponentRef | null>(null)

    // ── Computed ──
    // 画像はbase64で入っているので、データURIへの変換は computed にする。
    // テンプレートで直接呼ぶと再レンダーのたびに文字列全体を連結し直すことになり、
    // サムネイルは1件あたり平均406KB・最大10MBあるため無視できない。
    const favicon_src = computed(() => {
        const image = props.kyou.typed_urlog?.favicon_image
        return image ? base64_to_data_uri(image) : noimage
    })

    const thumbnail_src = computed(() => {
        const image = props.kyou.typed_urlog?.thumbnail_image
        return image ? base64_to_data_uri(image) : noimage
    })

    // ── Business logic ──
    function base64_to_data_uri(base64: string): string {
        if (base64.startsWith('/9j/')) return 'data:image/jpeg;base64,' + base64
        if (base64.startsWith('iVBOR')) return 'data:image/png;base64,' + base64
        if (base64.startsWith('R0lG')) return 'data:image/gif;base64,' + base64
        if (base64.startsWith('UklG')) return 'data:image/webp;base64,' + base64
        return 'data:image/png;base64,' + base64
    }

    function show_context_menu(e: PointerEvent): void {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    function open_urlog_link(): void {
        const url = props.kyou.typed_urlog?.url
        if (url) {
            window.open(url, "_blank")
        }
    }

    // ── Event relay objects ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Return ──
    return {
        // Template refs
        context_menu,

        // Computed
        favicon_src,
        thumbnail_src,

        // Business logic
        show_context_menu,
        open_urlog_link,

        // Event relay objects
        crudRelayHandlers,
    }
}

