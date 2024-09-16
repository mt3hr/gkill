<template>
    <v-dialog v-model="is_show_dialog">
        <EditMiView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="cloned_kyou" :last_added_tag="last_added_tag"
            :mi="cloned_kyou.typed_mi" @received_errors="(errors) => emits('received_errors', errors)"
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
        <NewBoardNameDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @setted_new_board_name="(new_board_name) => cloned_kyou.typed_mi.board_name = new_board_name" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref, watch } from 'vue';
import type { EditMiDialogProps } from './edit-mi-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditMiView from '../views/edit-mi-view.vue';
import KyouView from '../views/kyou-view.vue';
import NewBoardNameDialog from './new-board-name-dialog.vue';
import type { Kyou } from '@/classes/datas/kyou';

const props = defineProps<EditMiDialogProps>();
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
    errors = await cloned_kyou.value.load_typed_mi()
    if (errors && errors.length !== 0) {
        emits('received_errors', errors)
    }
}
</script>
