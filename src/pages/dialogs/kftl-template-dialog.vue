<template>
    <v-dialog v-model="is_show_dialog">
    <kFTLTemplateDialog />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import type { KFTLTemplateDialogEmits } from './kftl-template-dialog-emits'
import type { KFTLTemplateDialogProps } from './kftl-template-dialog-props'

const props = defineProps<KFTLTemplateDialogProps>()
const emits = defineEmits<KFTLTemplateDialogEmits>()
const child_templates: Ref<Array<KFTLTemplateElementData>> = ref(new Array<KFTLTemplateElementData>())
const is_show_child_dialog: Ref<boolean> = ref(false)

const is_show_dialog: Ref<boolean> = ref(false)

async function clicked_template_button(template_element: KFTLTemplateElementData): Promise<void> {
    throw new Error('Not implemented')
}

async function emit_clicked_template_element_leaf(template_leaf: KFTLTemplateElementData): Promise<void> {
    throw new Error('Not implemented')
}

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
