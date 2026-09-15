import { computed, onMounted, onUnmounted, ref, type Ref } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useRoute } from 'vue-router'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import type { SharedMiPageProps } from '@/pages/shared-mi-page-props'
import { useGkillMessageFeed } from '@/classes/use-gkill-message-feed'

export function useSharedMiPage(options: {
    props: SharedMiPageProps
}) {
    const _props = options.props

    // ── State refs ──
    const actual_height: Ref<number> = ref(0)
    const element_height: Ref<number> = ref(0)
    const browser_url_bar_height: Ref<number> = ref(0)
    const app_title_bar_height: Ref<number> = ref(50)
    const app_content_height: Ref<number> = ref(0)
    const app_content_width: Ref<number> = ref(0)
    // share_id が無くても throw しないこと（`!` を付けると undefined.toString() で
    // setup ごと失敗し、エラーも出ない真っ白な画面になる）。
    // use-shared-page.ts と同じ受け方に揃えてある
    const share_kyou_id = computed(() => useRoute().query.share_id?.toString() ?? '')

    const { push_errors, push_messages } = useGkillMessageFeed()

    function write_errors(errors_: Array<GkillError>): void {
        push_errors(errors_)
    }

    function write_messages(messages_: Array<GkillMessage>): void {
        push_messages(messages_)
    }

    // ── Helpers ──

    function resize_content(): void {
        const inner_element = document.querySelector('#control-height')
        actual_height.value = window.innerHeight
        element_height.value = inner_element ? inner_element.clientHeight : actual_height.value
        browser_url_bar_height.value = (Number(element_height.value) - Number(actual_height.value)).valueOf()
        app_content_height.value = (Number(element_height.value) - (Number(browser_url_bar_height.value) + Number(app_title_bar_height.value))).valueOf()
        app_content_width.value = window.innerWidth
    }

    // ── Lifecycle ──
    const onResize = () => {
        resize_content()
    }
    window.addEventListener('resize', onResize)
    onMounted(async () => {
        await reset_dialog_history()
    })
    onUnmounted(() => {
        window.removeEventListener('resize', onResize)
    })

    // ── Init ──
    resize_content()

    return {
        // State
        actual_height,
        app_title_bar_height,
        app_content_height,
        app_content_width,
        share_kyou_id,

        // Event handlers
        write_errors,
        write_messages,
    }
}
