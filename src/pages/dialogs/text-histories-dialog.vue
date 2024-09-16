<template>
    <v-dialog v-model="is_show_dialog">
        <KyouView :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
            :is_image_view="false" :kyou="kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
            :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
            :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)" />
        <TextHistoriesView :application_config="application_config" :gkill_api="gkill_api" :text="text"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import type { TextHistoriesDialogProps } from './text-histories-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import type { Kyou } from '@/classes/datas/kyou';
import { Text } from '@/classes/datas/text';
import KyouView from '../views/kyou-view.vue';
import TextHistoriesView from '../views/text-histories-view.vue';

const props = defineProps<TextHistoriesDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
defineExpose({ show, hide })

const cloned_text: Ref<Text> = ref(await props.text.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone())
const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>