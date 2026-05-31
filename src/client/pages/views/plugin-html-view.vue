<template>
    <div class="plugin-html-view" @contextmenu.prevent="show_context_menu">
        <div v-if="is_loading" class="plugin-loading">
            <v-progress-circular indeterminate />
        </div>
        <div v-else-if="error_message" class="plugin-error">
            {{ error_message }}
        </div>
        <!-- プラグインのHTMLをiframeのsrcdocで表示。
             sandbox: allow-same-originを付けないことでセッションcookieにアクセスさせない。
             高さはiframe内コンテンツからのpostMessageで動的に決定し、
             スクロールは親コンポーネントに任せる。 -->
        <iframe
            v-else-if="html"
            ref="iframe_ref"
            :srcdoc="html"
            sandbox="allow-scripts allow-forms"
            class="plugin-content-iframe"
            scrolling="no"
            :style="{
                width: '100%',
                height: iframe_height,
                'pointer-events': allow_pointer_events ? 'auto' : 'none',
                overflow: 'hidden',
            }"
            @load="on_iframe_load"
        />
        <PluginHtmlContextMenu
            :application_config="application_config"
            :gkill_api="gkill_api"
            :highlight_targets="highlight_targets"
            :kyou="kyou"
            :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers"
            ref="context_menu" />
    </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import PluginHtmlContextMenu from './plugin-html-context-menu.vue'
import type { PluginHtmlViewProps } from './plugin-html-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { usePluginHtmlView } from '@/classes/use-plugin-html-view'
import { GkillAPI } from '../../classes/api/gkill-api'
import { GetPluginContentHTMLRequest } from '../../classes/api/req_res/get-plugin-content-html-request'

const props = defineProps<PluginHtmlViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    show_context_menu,
    crudRelayHandlers,
} = usePluginHtmlView({ props, emits })

const html = ref<string>('')
const is_loading = ref<boolean>(true)
const error_message = ref<string>('')
const iframe_ref = ref<HTMLIFrameElement | null>(null)

// iframeコンテンツからpostMessageで受け取ったコンテンツ高さ（px）
const iframe_content_height = ref<number>(0)

// コンテンツ高さが確定するまでのフォールバック高さ
const iframe_height = computed<string>(() =>
    iframe_content_height.value > 0 ? iframe_content_height.value + 'px' : '80px'
)

// リストコンテキスト（height が数値）ではクリック不可、それ以外は操作可能
const allow_pointer_events = computed<boolean>(() => typeof props.height !== 'number')

// iframeにテーマをpostMessageで通知する
function send_theme_to_iframe(): void {
    const theme = props.application_config.use_dark_theme ? 'dark' : 'light'
    iframe_ref.value?.contentWindow?.postMessage({ gkill_theme: theme }, '*')
}

function on_iframe_load(): void {
    send_theme_to_iframe()
}

// iframeからのpostMessageを受信してコンテンツサイズを反映
function on_window_message(e: MessageEvent): void {
    // 自分のiframe以外からのメッセージは無視
    if (!iframe_ref.value || e.source !== iframe_ref.value.contentWindow) return
    if (e.data && e.data.gkill_iframe_size) {
        const h = e.data.gkill_iframe_size.height
        if (typeof h === 'number' && h > 0) {
            iframe_content_height.value = h
        }
    }
}

// テーマ変更を監視してiframeに通知
watch(() => props.application_config.use_dark_theme, () => {
    send_theme_to_iframe()
})

onMounted(async () => {
    window.addEventListener('message', on_window_message)

    if (!props.kyou.typed_plugin) {
        is_loading.value = false
        return
    }

    const req = new GetPluginContentHTMLRequest()
    req.session_id = GkillAPI.get_gkill_api().get_session_id()
    req.rep_name = props.kyou.typed_plugin.rep_name
    req.kyou_id = props.kyou.id

    const res = await GkillAPI.get_gkill_api().get_plugin_content_html(req)
    is_loading.value = false

    if (res.errors && res.errors.length > 0) {
        error_message.value = res.errors.map(e => e.error_message).join(', ')
        return
    }
    html.value = res.html
})

onUnmounted(() => {
    window.removeEventListener('message', on_window_message)
})

defineExpose({ show_context_menu })
</script>

<style>
.plugin-html-view {
    width: 100%;
}
.plugin-content-iframe {
    border: none;
    display: block;
}
.plugin-loading,
.plugin-error {
    padding: 8px;
    font-size: 0.85em;
    color: gray;
}
</style>
