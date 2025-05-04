<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog" class="kyou_dialog">
        <v-card class="pa-2" :width="'fit-content'">
            <KyouListView :kyou_height="180" :width="400" :list_height="list_height"
                :application_config="application_config" :gkill_api="gkill_api" :matched_kyous="model_value!"
                :query="new FindKyouQuery()" :last_added_tag="last_added_tag" :is_focused_list="true" :closable="false"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="true"
                :show_checkbox="true" :show_footer="true" :is_show_doc_image_toggle_button="true"
                :is_show_arrow_button="true" :show_content_only="false" :show_timeis_plaing_end_button="false"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" ref="kyou_list_views"
                @deleted_kyou="(deleted_kyou) => { /* reload_list(index); reload_kyou(deleted_kyou); */ }"
                @deleted_tag="(deleted_tag) => { }" @deleted_text="(deleted_text) => { }"
                @deleted_notification="(deleted_notification) => { }" @registered_kyou="(registered_kyou) => { }"
                @registered_tag="(registered_tag) => { }" @registered_text="(registered_text) => { }"
                @registered_notification="(registered_notification) => { }"
                @updated_kyou="(updated_kyou) => reload_kyou(updated_kyou)" @updated_tag="(updated_tag) => { }"
                @updated_text="(updated_text) => { }" @updated_notification="(updated_notification) => { }" />

        </v-card>
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import KyouListView from '../views/kyou-list-view.vue';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import type { Kyou } from '@/classes/datas/kyou';
import type { KyouListViewDialogProps } from './kyou-list-view-dialog-props';
import type { KyouListViewEmits } from '../views/kyou-list-view-emits';

defineProps<KyouListViewDialogProps>()
const model_value = defineModel<Array<Kyou>>()
const emits = defineEmits<KyouListViewEmits>()

defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}

async function reload_kyou(kyou: Kyou): Promise<void> {
    (async (): Promise<void> => {
        const kyous_list = model_value.value!
        for (let j = 0; j < kyous_list.length; j++) {
            const kyou_in_list = kyous_list[j]
            if (kyou.id === kyou_in_list.id) {
                const updated_kyou = kyou.clone()
                await updated_kyou.reload()
                await updated_kyou.load_all()
                kyous_list.splice(j, 1, updated_kyou)
                break
            }
        }
    })();
}
</script>
