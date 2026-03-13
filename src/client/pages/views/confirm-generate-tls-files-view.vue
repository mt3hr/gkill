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
import type { ConfirmGenerateTLSFilesViewEmits } from './confirm-generate-tls-files-view-emits'
import type { ConfirmGenerateTLSFilesViewProps } from './confirm-generate-tls-files-view-props'
import { useConfirmGenerateTlsFilesView } from '@/classes/use-confirm-generate-tls-files-view'

const props = defineProps<ConfirmGenerateTLSFilesViewProps>()
const emits = defineEmits<ConfirmGenerateTLSFilesViewEmits>()

const {
    generate_tls_files,
} = useConfirmGenerateTlsFilesView({ props, emits })
</script>
