<template>
    <v-dialog v-model="is_show_dialog">
        <EditLantanaView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="cloned_kyou" :lantana="cloned_kyou.typed_lantana"
            :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)" />
        <KyouView :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
            :is_image_view="false" :kyou="cloned_kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
            :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
            :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref, watch } from 'vue';
import type { EditLantanaDialogProps } from './edit-lantana-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditLantanaView from '../views/edit-lantana-view.vue';
import KyouView from '../views/kyou-view.vue';
import type { Kyou } from '@/classes/datas/kyou';

const props = defineProps<EditLantanaDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })
watch(props.kyou, () => update_cloned_kyou())

const is_show_dialog: Ref<boolean> = ref(false)
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
async function update_cloned_kyou(): Promise<void> {
    let errors
    cloned_kyou.value = props.kyou
    errors = await cloned_kyou.value.load_typed_lantana()
    if (errors && errors.length !== 0) {
        emits('received_errors', errors)
    }
}
</script>
