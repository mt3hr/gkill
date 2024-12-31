<template>
    <v-card>
        <v-card-title>
            記録保管場所編集
        </v-card-title>
        <p>{{ struct_obj.rep_name }}</p>
        <v-checkbox v-model="check_when_inited" hide-detail label="初期化時チェック" />
        <v-checkbox v-model="ignore_check_rep_rykv" hide-detail label="サマリチェック無効" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="apply">適用</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="emits('requested_close_dialog')">キャンセル</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditRepStructElementViewEmits } from './edit-rep-struct-element-view-emits'
import type { EditRepStructElementViewProps } from './edit-rep-struct-element-view-props'
import { RepStruct } from '@/classes/datas/config/rep-struct';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import { GkillAPI } from '@/classes/api/gkill-api';

const props = defineProps<EditRepStructElementViewProps>()
const emits = defineEmits<EditRepStructElementViewEmits>()

const check_when_inited: Ref<boolean> = ref(props.struct_obj.check_when_inited)
const ignore_check_rep_rykv: Ref<boolean> = ref(props.struct_obj.ignore_check_rep_rykv)

async function apply(): Promise<void> {
    const gkill_info_req = new GetGkillInfoRequest()
    gkill_info_req.session_id = props.gkill_api.get_session_id()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }
    if (gkill_info_res.messages && gkill_info_res.messages.length !== 0) {
        emits('received_messages', gkill_info_res.messages)
    }

    const rep_struct = new RepStruct()
    rep_struct.id = props.struct_obj.id
    rep_struct.device = gkill_info_res.device
    rep_struct.user_id = gkill_info_res.user_id
    rep_struct.rep_name = props.struct_obj.rep_name
    rep_struct.parent_folder_id = props.struct_obj.parent_folder_id
    rep_struct.check_when_inited = check_when_inited.value
    rep_struct.ignore_check_rep_rykv = ignore_check_rep_rykv.value
    rep_struct.seq = props.struct_obj.seq
    emits('requested_update_rep_struct', rep_struct)
}
</script>
