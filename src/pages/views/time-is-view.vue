<template>
    <TimeIsContextMenu :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
        :kyou="kyou" :last_added_tag="last_added_tag" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import type { TimeIsViewProps } from './time-is-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { TimeIs } from '@/classes/datas/time-is';
import TimeIsContextMenu from './time-is-context-menu.vue';

const props = defineProps<TimeIsViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_timeis: Ref<TimeIs> = ref(await props.timeis.clone());
</script>
