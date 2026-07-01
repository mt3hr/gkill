<template>
    <div class="plugin-html-view"
        :style="plugin_html_view_style"
        @contextmenu.prevent="show_context_menu">
        <div v-if="is_loading" class="plugin-loading">
            <v-progress-circular indeterminate />
        </div>
        <div v-else-if="error_message" class="plugin-error">
            {{ error_message }}
        </div>
        <!-- プラグインのHTMLをiframeで表示。
             sandbox: allow-same-originを付けないことでセッションcookieにアクセスさせない。
             高さはiframe内コンテンツからのpostMessageで動的に決定し、
             スクロールは親コンポーネントに任せる。
             v-show を使い iframe を常にDOMに残す。v-if/v-else-if でのDOM挿入は
             iOS/Android がフォーカス可能要素の挿入を検知してスクロール位置を自動変更する
             原因になるため、表示切り替えは display:none で行う。
             tabindex="-1" でフォーカス可能要素として扱われることも防ぐ。
             ダイアログ表示時: srcdocには定数ローダーを使用し、コンテンツはpostMessage経由で
             document.open()+document.write()で注入する（replacementナビゲーション）。
             これによりiframeナビゲーションがdialogのpushStateより前に完了し、
             ブラウザバック1回でダイアログが閉じるようになる。
             リスト表示時: 従来通りsrcdocを直接更新する。 -->
        <iframe
            v-show="!!html && !is_loading && !error_message"
            tabindex="-1"
            ref="iframe_ref"
            :srcdoc="effective_srcdoc"
            sandbox="allow-scripts allow-forms"
            class="plugin-content-iframe"
            scrolling="no"
            :style="{
                width: '100%',
                height: iframe_height,
                'pointer-events': allow_pointer_events ? 'auto' : 'none',
                overflow: 'hidden',
                'overflow-anchor': 'none',
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
import { i18n } from '@/i18n'
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

// ダイアログ表示時にiframeのsrcdocナビゲーションがpushStateより後にならないよう、
// srcdocには定数ローダーを使い、コンテンツはpostMessageで注入する。
// ローダーはgkill_plugin_htmlメッセージを受け取るとdocument.open()+write()+close()で
// コンテンツを差し替える（replacementナビゲーション = joint historyエントリを増やさない）。
const PLUGIN_IFRAME_LOADER = '<!DOCTYPE html><html><head><meta charset="utf-8"><style>html,body{margin:0;padding:0;background:transparent}</style></head><body><script>(function(){window.addEventListener("message",function(e){if(!e.data||typeof e.data.gkill_plugin_html!=="string")return;document.open();document.write(e.data.gkill_plugin_html);document.close();});})();<\/script></body></html>'

const html = ref<string>('')
const is_loading = ref<boolean>(true)
const error_message = ref<string>('')
const iframe_ref = ref<HTMLIFrameElement | null>(null)
// 最後にiframeに注入したHTML。重複注入・無限ループを防ぐ。
const sent_html = ref<string>('')

const plugin_html_view_style = computed((): Record<string, string> => {
    if (typeof props.height !== 'number') {
        return {}
    }
    return {
        height: props.height + 'px',
        overflow: 'hidden',
        contain: 'layout',
        'overflow-anchor': 'none',
    }
})

// iframeコンテンツからpostMessageで受け取ったコンテンツ高さ（px）
const iframe_content_height = ref<number>(0)

// iframeの表示高さを計算する。
// リストコンテキスト（height が数値＝KyouListView内）では props.height を固定値として返す。
// postMessageによる動的な高さ変化を許容すると、VVirtualScrollItemのResizeObserverが
// 変化を検出してoffsetを再計算しスクロール位置がずれるため。
const iframe_height = computed<string>(() => {
    if (typeof props.height === 'number') {
        return props.height + 'px'
    }
    return iframe_content_height.value > 0 ? iframe_content_height.value + 'px' : '80px'
})

// リストコンテキスト（height が数値）ではクリック不可、それ以外は操作可能
const allow_pointer_events = computed<boolean>(() => typeof props.height !== 'number')

// リストコンテキスト判定（height が数値 = KyouListView内）
// リスト表示時は従来通りsrcdocを直接使用し、
// ダイアログ表示時は定数ローダー + postMessage注入方式を使用する。
const is_list_view = computed<boolean>(() => typeof props.height === 'number')

// ダイアログ表示: 定数ローダー（変更されないのでiframeの再ナビゲーションが発生しない）
// リスト表示: 従来通りhtmlを直接srcdocに設定
const effective_srcdoc = computed<string | undefined>(() => {
    if (is_list_view.value) {
        return html.value || undefined
    }
    return PLUGIN_IFRAME_LOADER
})

// iframeにテーマをpostMessageで通知する
function send_theme_to_iframe(): void {
    const theme = props.application_config.use_dark_theme ? 'dark' : 'light'
    iframe_ref.value?.contentWindow?.postMessage({ gkill_theme: theme }, '*')
}

// ダイアログ表示時: htmlが用意できたらpostMessageでiframeローダーに注入する。
// sent_htmlで重複チェックし、@loadが繰り返し発火しても無限ループにならないようにする。
function try_inject_html(): void {
    if (is_list_view.value) return
    if (!html.value) return
    if (html.value === sent_html.value) return
    if (!iframe_ref.value?.contentWindow) return
    sent_html.value = html.value
    iframe_ref.value.contentWindow.postMessage({ gkill_plugin_html: html.value }, '*')
}

function on_iframe_load(): void {
    try_inject_html()
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

// ダイアログ表示時: htmlが変化したらiframeに注入を試みる
// （ローダーがすでにロード済みの場合に即座に反映させるため）
watch(html, () => {
    try_inject_html()
})

// HTMLコンテンツを取得してセットする。
// 開始時点のkyou.idを保持し、レスポンス受信後に現在のprops.kyou.idと
// 一致しない場合は結果を破棄することでレース条件を防止する。
async function load_html(): Promise<void> {
    const target_id = props.kyou.id

    html.value = ''
    sent_html.value = ''
    iframe_content_height.value = 0
    is_loading.value = true
    error_message.value = ''

    if (!props.kyou.typed_plugin) {
        is_loading.value = false
        return
    }

    const req = new GetPluginContentHTMLRequest()
    req.session_id = GkillAPI.get_gkill_api().get_session_id()
    req.rep_name = props.kyou.typed_plugin.rep_name
    req.kyou_id = props.kyou.id

    try {
        const res = await GkillAPI.get_gkill_api().get_plugin_content_html(req)

        // レスポンス到着時点でkyouが別のものに変わっていたら無視
        if (props.kyou.id !== target_id) {
            return
        }

        is_loading.value = false

        if (res.errors && res.errors.length > 0) {
            error_message.value = res.errors.map(e => e.error_message).join(', ')
            return
        }
        html.value = res.html
    } catch (e: unknown) {
        // kyouが変わっていたら別のload_html()に委ねる
        if (props.kyou.id !== target_id) {
            return
        }
        is_loading.value = false
        error_message.value = e instanceof Error ? e.message : i18n.global.t('PLUGIN_CONTENT_FETCH_FAILED_MESSAGE')
    }
}

// v-virtual-scrollによるコンポーネント再利用時もHTMLを再ロードするためkyou.idを監視する。
// immediate: trueにより初回マウント時もこのwatchでHTMLをロードする。
watch(() => props.kyou.id, async () => {
    await load_html()
}, { immediate: true })

// messageリスナー登録のみ。HTMLロードはwatchのimmediate:trueに委ねる。
onMounted(() => {
    window.addEventListener('message', on_window_message)
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
