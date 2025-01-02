<template>
    <v-dialog v-model="is_show_dialog" :width="'fit-content'">
        <KFTLTemplateView :application_config="application_config" :gkill_api="gkill_api" :template="template"
            @clicked_template_element_leaf="(template: KFTLTemplateElementData) => emits('clicked_template_element_leaf', template)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide()" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import type { KFTLTemplateDialogEmits } from './kftl-template-dialog-emits'
import type { KFTLTemplateDialogProps } from './kftl-template-dialog-props'
import KFTLTemplateView from '../views/kftl-template-view.vue'

defineProps<KFTLTemplateDialogProps>()
const emits = defineEmits<KFTLTemplateDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    emits('closed_dialog')
}
</script>
