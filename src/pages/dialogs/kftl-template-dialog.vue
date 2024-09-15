<template>
    <kFTLTemplateDialog />
</template>
<script lang="ts" setup>
import { KFTLTemplateElement } from '@/classes/datas/kftl-template-element';
import type { KFTLTemplateDialogEmits } from './kftl-template-dialog-emits';
import type { KFTLTemplateDialogProps } from './kftl-template-dialog-props';
import { ref, type Ref } from 'vue';

const props = defineProps<KFTLTemplateDialogProps>();
const emits = defineEmits<KFTLTemplateDialogEmits>();
const child_templates: Ref<Array<KFTLTemplateElement>> = ref(new Array<KFTLTemplateElement>());
const is_show_child_dialog: Ref<boolean> = ref(false);

async function clicked_template_button(template_element: KFTLTemplateElement): Promise<void> {
    throw new Error('Not implemented');
}

async function emit_clicked_template_element_leaf(template_leaf: KFTLTemplateElement): Promise<void> {
    throw new Error('Not implemented');
}

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>