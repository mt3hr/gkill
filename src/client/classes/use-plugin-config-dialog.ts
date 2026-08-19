'use strict'

import { ref, watch, onMounted, onUnmounted, type ModelRef } from 'vue'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetPluginConfigHTMLRequest } from '@/classes/api/req_res/get-plugin-config-html-request'
import { PostPluginConfigRequest } from '@/classes/api/req_res/post-plugin-config-request'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import type { PluginConfigDialogProps } from '@/pages/dialogs/plugin-config-dialog-props'

export function usePluginConfigDialog(options: {
    props: PluginConfigDialogProps
    // 開閉は v-model（defineModel）なので、.vue 側で作ったモデルを受け取る。
    // defineModel はコンパイラマクロで composable の中では書けない
    show: ModelRef<boolean>
}) {
    const { props, show } = options

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

    return {
        html,
        is_loading,
        error_message,
        iframe_ref,
        send_theme_to_iframe,
    }
}
