<template>
    <v-card :max-width="500" class="help_card pa-3">
        <v-card-title>
            <v-row>
                <v-col cols="auto" class="pa-0 ma-0">
                    {{ i18n.global.t("KFTL_TEMPLATE_TITLE") }}
                </v-col>
                <v-spacer />
                <v-col class="pa-0 ma-0" cols="auto">
                    <v-btn icon="mdi-close" @click="emits('requested_close_dialog')" />
                </v-col>
            </v-row>
        </v-card-title>
        <div class="button_list">
            <v-btn dark color="primary" class="pa-3 ma-3" v-for="template, index in template.children"
                :key="template.id!" @click="clicked_template_button(template, index)">
                {{ template.title }}
                <KFTLTemplateDialog :application_config="application_config" :gkill_api="gkill_api" :template="template"
                    @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                    @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                    @clicked_template_element_leaf="(...template: any[]) => emits('clicked_template_element_leaf', template[0] as KFTLTemplateElementData)"
                    @requested_close_dialog="emits('requested_close_dialog')" ref="child_template_dialogs" />
            </v-btn>
        </div>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data';
import KFTLTemplateDialog from '../dialogs/kftl-template-dialog.vue';
import type { KFTLTemplateViewProps } from './kftl-template-view-props';
import type { KFTLTemplateViewEmits } from './kftl-template-view-emits';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

const child_template_dialogs: Ref<Array<any>> = ref(new Array<any>())

defineProps<KFTLTemplateViewProps>()
const emits = defineEmits<KFTLTemplateViewEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

function clicked_template_button(template: KFTLTemplateElementData, index: number): void {
    if (!template.children) {
        emits('clicked_template_element_leaf', template)
        emits('requested_close_dialog')
        return
    }
    child_template_dialogs.value[index].show()
}

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
