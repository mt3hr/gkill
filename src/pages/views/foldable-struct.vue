<template></template>
<script lang="ts" setup>
import { ref, type Ref } from 'vue';
import type { FoldableStructEmits } from './foldable-struct-emits';
import type { FoldableStructProps } from './foldable-struct-props';
import { CheckState } from './check-state';

const props = defineProps<FoldableStructProps>();
const emits = defineEmits<FoldableStructEmits>();
defineExpose({ get_selected_items })

const cloned_is_open: Ref<boolean> = ref(props.is_open);
const check: Ref<CheckState> = ref(CheckState.checked);
const struct_list: Ref<any> = ref(props.struct_obj);

async function get_selected_items(): Promise<Array<string>> {
    throw new Error('Not implemented');
}
</script>