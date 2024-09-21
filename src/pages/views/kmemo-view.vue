<template>
    <KmemoContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="cloned_kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import KmemoContextMenu from './kmemo-context-menu.vue';
import type { Kmemo } from '@/classes/datas/kmemo';
import type { KmemoViewProps } from './kmemo-view-props.ts';
import type { KyouViewEmits } from './kyou-view-emits';
import type { Kyou } from '@/classes/datas/kyou';

const props = defineProps<KmemoViewProps>();
const emits = defineEmits<KyouViewEmits>();

const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
const cloned_kmemo: Ref<Kmemo> = ref(await props.kmemo.clone());
</script>
