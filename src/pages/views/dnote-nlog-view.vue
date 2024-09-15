<template>
    <KyouDialog />
</template>
<script setup lang="ts">
import type { DnoteNlogViewEmits } from './dnote-nlog-view-emits';
import type { DnoteNlogViewProps } from './dnote-nlog-view-props';
import KyouDialog from '../dialogs/kyou-dialog.vue';
import { type Ref, ref, computed } from 'vue';
import { Kyou } from '@/classes/datas/kyou';

const props = defineProps<DnoteNlogViewProps>();
const emits = defineEmits<DnoteNlogViewEmits>();
const cloned_kyou: Ref<Kyou> = ref(await props.nlog_kyou.clone());
const cloned_nlog = computed(async () => {
    const errors = await cloned_kyou.value.load_typed_nlog()
    if (errors && errors.length !== 0) {
        emits('received_errors', errors)
        return
    }
    const nlog = cloned_kyou.value.typed_nlog
    return nlog
})
</script>
