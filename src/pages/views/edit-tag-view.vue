<template>
    <KyouView :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
        :is_image_view="false" :kyou="cloned_kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
        :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
        :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditTagViewProps } from './edit-tag-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import KyouView from './kyou-view.vue';
import { Tag } from '@/classes/datas/tag';
import { Kyou } from '@/classes/datas/kyou';

const props = defineProps<EditTagViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_tag: Ref<Tag> = ref(await props.tag.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>
