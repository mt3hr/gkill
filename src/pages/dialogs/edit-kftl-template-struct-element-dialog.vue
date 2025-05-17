<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditKFTLTemplateStructElementView :application_config="application_config" :gkill_api="gkill_api"
            :struct_obj="kftl_template_struct" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_kftl_template_struct="(kftl_template_struct) => emits('requested_update_kftl_template_struct', kftl_template_struct)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { EditKFTLTemplateStructElementDialogEmits } from './edit-kftl-template-struct-element-dialog-emits'
import type { EditKFTLTemplateStructElementDialogProps } from './edit-kftl-template-struct-element-dialog-props'
import EditKFTLTemplateStructElementView from '../views/edit-kftl-template-struct-element-view.vue'
import { KFTLTemplateStruct } from '@/classes/datas/config/kftl-template-struct';

defineProps<EditKFTLTemplateStructElementDialogProps>()
const emits = defineEmits<EditKFTLTemplateStructElementDialogEmits>()
defineExpose({ show, hide })

const kftl_template_struct: Ref<KFTLTemplateStruct> = ref(new KFTLTemplateStruct())
const is_show_dialog: Ref<boolean> = ref(false)

async function show(kftl_template_struct_obj: KFTLTemplateStruct): Promise<void> {
    kftl_template_struct.value = kftl_template_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
