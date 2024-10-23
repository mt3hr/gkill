<template>
    <v-card @contextmenu.prevent="show_context_menu">
        <KyouView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="[kyou.generate_info_identifer()]" :is_image_view="false" :kyou="target_kyou"
            :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
            :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
            :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
            @dblclick.prevent.stop="() => {}" />
    </v-card>
    <ReKyouContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[kyou.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag"
        ref="context_menu" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import type { ReKyouViewProps } from './re-kyou-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import ReKyouContextMenu from './re-kyou-context-menu.vue'
import { computed, type Ref, ref } from 'vue'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import { ReKyou } from '@/classes/datas/re-kyou'
import { Kyou } from '@/classes/datas/kyou'
import KyouView from './kyou-view.vue'

const context_menu = ref<InstanceType<typeof ReKyouContextMenu> | null>(null);

const props = defineProps<ReKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const target_kyou: Ref<Kyou> = ref(new Kyou())

async function get_target_kyou() {
    const req = new GetKyouRequest()
    req.id = props.rekyou.target_id
    const res = await props.gkill_api.get_kyou(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    target_kyou.value = res.kyou_histories[0]
}

function show_context_menu(e: PointerEvent): void {
    context_menu.value?.show(e)
}

get_target_kyou()
</script>
