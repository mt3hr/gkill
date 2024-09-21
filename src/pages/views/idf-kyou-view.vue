<template>
    <IDFKyouContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="cloned_kyou" :last_added_tag="last_added_tag"
        :idf_kyou="cloned_idf_kyou" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import IDFKyouContextMenu from './idf-kyou-context-menu.vue';
import type { IDFKyou } from '@/classes/datas/idf-kyou';
import { type Ref, ref } from 'vue';
import type { IDFKyouProps } from './idf-kyou-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { Kyou } from '@/classes/datas/kyou';

const props = defineProps<IDFKyouProps>();
const emits = defineEmits<KyouViewEmits>();

const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
const cloned_idf_kyou: Ref<IDFKyou> = ref(await props.idf_kyou.clone());
</script>
