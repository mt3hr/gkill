<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>TLSファイル生成</span>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card>
            <p>下記ファイルにTLS用ファイルを生成します。</p>
            <p>すでに存在する場合は上書きされます）</p>
            <table>
                <tr>
                    <td>
                        TLS CERTファイル：
                    </td>
                    <td>
                        {{ server_config.tls_cert_file }}
                    </td>
                </tr>
                <tr>
                    <td>
                        TLS KEYファイ：
                    </td>
                    <td>
                        {{ server_config.tls_key_file }}
                    </td>
                </tr>
            </table>
        </v-card>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="generate_tls_files">作成</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="emits('requested_close_dialog')">キャンセル</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { GenerateTLSFileRequest } from '@/classes/api/req_res/generate-tls-file-request';
import type { ConfirmGenerateTLSFilesViewEmits } from './confirm-generate-tls-files-view-emits'
import type { ConfirmGenerateTLSFilesViewProps } from './confirm-generate-tls-files-view-props'
import type { GenerateTLSFileResponse } from '@/classes/api/req_res/generate-tls-file-response';

const props = defineProps<ConfirmGenerateTLSFilesViewProps>()
const emits = defineEmits<ConfirmGenerateTLSFilesViewEmits>()

async function generate_tls_files(): Promise<void> {
    const req = new GenerateTLSFileRequest()
    req.session_id = props.gkill_api.get_session_id()
    const res: GenerateTLSFileResponse = await props.gkill_api.generate_tls_file(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('generated_tls_files')
}
</script>
