<template>
    <v-card>
        <v-card-title>
            デバイス編集
        </v-card-title>
        <p>{{ struct_obj.device_name }}</p>
        <v-checkbox v-model="check_when_inited" hide-detail label="初期化時チェック" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="apply" color="primary">適用</v-btn>
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
import type { EditDeviceStructElementViewEmits } from './edit-device-struct-element-view-emits'
import type { EditDeviceStructElementViewProps } from './edit-device-struct-element-view-props'
import { DeviceStruct } from '@/classes/datas/config/device-struct';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';

const props = defineProps<EditDeviceStructElementViewProps>()
const emits = defineEmits<EditDeviceStructElementViewEmits>()

const check_when_inited: Ref<boolean> = ref(props.struct_obj.check_when_inited)

async function apply(): Promise<void> {
    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }
    if (gkill_info_res.messages && gkill_info_res.messages.length !== 0) {
        emits('received_messages', gkill_info_res.messages)
    }

    const device_struct = new DeviceStruct()
    device_struct.id = props.struct_obj.id
    device_struct.device = gkill_info_res.device
    device_struct.user_id = gkill_info_res.user_id
    device_struct.device_name = props.struct_obj.device_name
    device_struct.parent_folder_id = props.struct_obj.parent_folder_id
    device_struct.check_when_inited = check_when_inited.value
    device_struct.seq = props.struct_obj.seq
    emits('requested_update_device_struct', device_struct)
}
</script>
