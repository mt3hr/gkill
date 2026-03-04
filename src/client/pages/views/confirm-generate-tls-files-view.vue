<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t('GENERATE_OREORE_TLS_TITLE') }}</span>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card>
            <p>
            <pre>{{ i18n.global.t("GENERATE_OREORE_TLS_FILE_MESSAGE") }}</pre>
            </p>
            <table>
                <tr>
                    <td>
                        {{ i18n.global.t("TLS_CERT_FILE_TITLE") }}：
                    </td>
                    <td>
                        {{ server_config.tls_cert_file }}
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("TLS_KEY_FILE_TITLE") }}：
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
                    <v-btn dark @click="generate_tls_files" color="primary">{{ i18n.global.t("CREATE_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ i18n.global.t("CANCEL_TITLE")
                        }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { GenerateTLSFileRequest } from '@/classes/api/req_res/generate-tls-file-request';
import type { ConfirmGenerateTLSFilesViewEmits } from './confirm-generate-tls-files-view-emits'
import type { ConfirmGenerateTLSFilesViewProps } from './confirm-generate-tls-files-view-props'
import type { GenerateTLSFileResponse } from '@/classes/api/req_res/generate-tls-file-response';

const props = defineProps<ConfirmGenerateTLSFilesViewProps>()
const emits = defineEmits<ConfirmGenerateTLSFilesViewEmits>()

async function generate_tls_files(): Promise<void> {
    const req = new GenerateTLSFileRequest()
    const res: GenerateTLSFileResponse = await props.gkill_api.generate_tls_file(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('generated_tls_files')
    emits('requested_close_dialog')
}
</script>
