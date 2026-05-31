<template>
    <v-dialog v-model="show" max-width="800">
        <v-card>
            <v-card-title>{{ rep_name }} 設定</v-card-title>
            <v-card-text>
                <div v-if="is_loading" class="d-flex justify-center pa-4">
                    <v-progress-circular indeterminate />
                </div>
                <div v-else-if="error_message" class="plugin-error">
                    {{ error_message }}
                </div>
                <!-- プラグインの設定フォームHTMLをiframeで表示。
                     allow-same-originを付けないことでセッションcookieを隔離。 -->
                <iframe
                    v-else-if="html"
                    :srcdoc="html"
                    sandbox="allow-scripts allow-forms"
                    class="plugin-config-iframe"
                />
            </v-card-text>
            <v-card-actions>
                <v-spacer />
                <v-btn @click="show = false">閉じる</v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { GkillAPI } from '../../classes/api/gkill-api'
import { GetPluginConfigHTMLRequest } from '../../classes/api/req_res/get-plugin-config-html-request'

const props = defineProps<{
    rep_name: string
}>()

const show = defineModel<boolean>('show', { default: false })

const html = ref<string>('')
const is_loading = ref<boolean>(false)
const error_message = ref<string>('')

watch(show, async (newVal) => {
    if (!newVal || !props.rep_name) return
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
