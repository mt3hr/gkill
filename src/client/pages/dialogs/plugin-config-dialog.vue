<template>
    <v-dialog v-model="show" max-width="800">
        <v-card variant="flat">
            <v-card-title>{{ rep_name }} {{ i18n.global.t('TOOLTIP_SETTINGS') }}</v-card-title>
            <v-card-text>
                <div v-if="is_loading" class="d-flex justify-center pa-4">
                    <v-progress-circular indeterminate />
                </div>
                <div v-else-if="error_message" class="plugin-error">
                    {{ error_message }}
                </div>
                <!-- プラグインの設定フォームHTMLをiframeで表示。
                     allow-same-originを付けないことでセッションcookieを隔離。
                     保存はフォーム送信ではなくpostMessageで親に依頼する（下の onWindowMessage）。 -->
                <iframe
                    v-else-if="html"
                    ref="iframe_ref"
                    :srcdoc="html"
                    sandbox="allow-scripts allow-forms"
                    class="plugin-config-iframe"
                    @load="send_theme_to_iframe"
                />
            </v-card-text>
            <v-card-actions class="gkill-dialog-actions">
                <v-spacer />
                <v-btn @click="show = false">{{ i18n.global.t('CLOSE_TITLE') }}</v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { i18n } from '@/i18n'
import { GkillAPI } from '../../classes/api/gkill-api'
import { GetPluginConfigHTMLRequest } from '../../classes/api/req_res/get-plugin-config-html-request'
import { PostPluginConfigRequest } from '../../classes/api/req_res/post-plugin-config-request'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

const props = defineProps<{
    rep_name: string
    application_config?: ApplicationConfig
}>()

const show = defineModel<boolean>('show', { default: false })
// iframe 内でプラグイン製フォームがナビゲーションし joint history entry を
// 作り得るため、close_dialog_via_history は使わずプログラム的クローズのままにする
// (unmount で iframe の履歴ごと消えてから巻き戻される)。登録だけ行い、
// ブラウザバック/Escape で閉じられるようにする。
useDialogHistoryStack(show)

const html = ref<string>('')
const is_loading = ref<boolean>(false)
const error_message = ref<string>('')
const iframe_ref = ref<HTMLIFrameElement | null>(null)

async function load_config_html(): Promise<void> {
    if (!props.rep_name) return
    is_loading.value = true
    error_message.value = ''
    html.value = ''

    const req = new GetPluginConfigHTMLRequest()
    req.session_id = GkillAPI.get_gkill_api().get_session_id()
    req.rep_name = props.rep_name

    const res = await GkillAPI.get_gkill_api().get_plugin_config_html(req)
    is_loading.value = false

    if (res.errors && res.errors.length > 0) {
        error_message.value = res.errors.map(e => e.error_message).join(', ')
        return
    }
    html.value = res.html
}

watch(show, async (new_val) => {
    if (!new_val) return
    await load_config_html()
})

// iframe にテーマを通知する。プラグイン側は data-theme 属性を切り替えて CSS 変数で追従する。
function send_theme_to_iframe(): void {
    const theme = props.application_config?.use_dark_theme ? 'dark' : 'light'
    iframe_ref.value?.contentWindow?.postMessage({ gkill_theme: theme }, '*')
}

// iframe から設定の保存を依頼される。
// sandbox に allow-same-origin を付けていない以上、iframe 内のフォームは
// 自力で gkill の API を叩けない。保存だけは親（ここ）が肩代わりする。
//   iframe → 親 : { gkill_plugin_config: { <key>: <value>, ... } }
//   親 → iframe : { gkill_plugin_config_result: { ok: boolean, error?: string } }
async function onWindowMessage(e: MessageEvent): Promise<void> {
    if (!iframe_ref.value || e.source !== iframe_ref.value.contentWindow) return
    const form = e.data?.gkill_plugin_config
    if (!form || typeof form !== 'object') return

    // 値は文字列だけ受け付ける（プラグインが何を送ってきても API の型を壊さない）
    const form_data: Record<string, string> = {}
    for (const [k, v] of Object.entries(form as Record<string, unknown>)) {
        form_data[String(k)] = typeof v === 'string' ? v : String(v)
    }

    const req = new PostPluginConfigRequest()
    req.session_id = GkillAPI.get_gkill_api().get_session_id()
    req.rep_name = props.rep_name
    req.form_data = form_data

    const res = await GkillAPI.get_gkill_api().post_plugin_config(req)
    const failed = res.errors && res.errors.length > 0
    const error_text = failed ? res.errors.map(err => err.error_message).join(', ') : undefined
    iframe_ref.value?.contentWindow?.postMessage({
        gkill_plugin_config_result: { ok: !failed, error: error_text },
    }, '*')

    if (failed) {
        error_message.value = error_text ?? ''
        return
    }
    // 保存後の状態（読み込み件数など）を反映するため取り直す
    await load_config_html()
}

onMounted(() => {
    window.addEventListener('message', onWindowMessage)
})
onUnmounted(() => {
    window.removeEventListener('message', onWindowMessage)
})
</script>

<style>
.plugin-config-iframe {
    width: 100%;
    height: 500px;
    border: none;
    display: block;
}
.plugin-error {
    color: red;
    padding: 8px;
}
</style>
