import { computed, onMounted, onUnmounted, ref, type Ref } from 'vue'
import { GkillAPI } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
// デフォルトインポートにするとpackage.json全体(依存一覧とoverrides込み)が
// バンドルに載ってしまうので、versionだけを名前付きで取る
import { version as package_version } from '../../../package.json'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import { useGkillMessageFeed } from '@/classes/use-gkill-message-feed'

export function useSetNewPasswordPage() {
    // ── State refs ──
    const actual_height: Ref<number> = ref(0)
    const element_height: Ref<number> = ref(0)
    const browser_url_bar_height: Ref<number> = ref(0)
    const app_title_bar_height: Ref<number> = ref(50)
    const app_title_bar_height_px = computed(() => app_title_bar_height.value.toString().concat("px"))
    const gkill_api = computed(() => GkillAPI.get_instance())
    const app_content_height: Ref<number> = ref(0)
    const app_content_width: Ref<number> = ref(0)
    const gkill_version: Ref<string> = ref(package_version)

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
        browser_url_bar_height.value = Number(element_height.value) - Number(actual_height.value)
        app_content_height.value = Number(element_height.value) - (Number(browser_url_bar_height.value) + Number(app_title_bar_height.value))
        app_content_width.value = window.innerWidth
    }

    // ── Event handlers ──
    function onReceivedErrors(errors: Array<GkillError>): void {
        write_errors(errors)
    }

    function onReceivedMessages(messages: Array<GkillMessage>): void {
        write_messages(messages)
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
        app_title_bar_height_px,
        gkill_api,
        app_content_height,
        app_content_width,
        gkill_version,

        // Event handlers
        onReceivedErrors,
        onReceivedMessages,
    }
}
