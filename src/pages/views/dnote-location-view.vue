<template>
    <KyouDialog />
</template>
<script setup lang="ts">
import type { DnoteLocationViewEmits } from './dnote-location-view-emits';
import type { DnoteLocationViewProps } from './dnote-location-view-props';
import KyouDialog from '../dialogs/kyou-dialog.vue';
import { type Ref, ref, computed } from 'vue';
import { Kyou } from '@/classes/datas/kyou';
import type { TimeIs } from '@/classes/datas/time-is';
import type { GkillError } from '@/classes/api/gkill-error';
import type { Kmemo } from '@/classes/datas/kmemo';

const props = defineProps<DnoteLocationViewProps>();
const emits = defineEmits<DnoteLocationViewEmits>();
const cloned_kyou: Ref<Array<Kyou>> = ref(props.timeis_or_kmemo_kyou.concat());
const cloned_timeis = computed(() => {
    const timeiss: Array<TimeIs> = new Array<TimeIs>()
    cloned_kyou.value.forEach(async (kyou) => {
        if (kyou.data_type !== "timeis") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_timeis()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        timeiss.push(kyou.typed_timeis)
    })
    return timeiss
})
const cloned_kmemo = computed(() => {
    const kmemos: Array<Kmemo> = new Array<Kmemo>()
    cloned_kyou.value.forEach(async (kyou) => {
        if (kyou.data_type !== "kmemo") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_kmemo()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        kmemos.push(kyou.typed_kmemo)
    })
    return kmemos
})
</script>