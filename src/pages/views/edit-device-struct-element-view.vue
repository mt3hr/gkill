<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("EDIT_DEVICE_TITLE") }}
        </v-card-title>
        <p>{{ struct_obj.device_name }}</p>
        <v-checkbox v-model="check_when_inited" hide-detail :label="i18n.global.t('CHECK_WHEN_INITED_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="apply" color="primary">{{ i18n.global.t('APPLY_TITLE') }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{
                        i18n.global.t('CANCEL_TITLE') }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue';
import type { EditDeviceStructElementViewEmits } from './edit-device-struct-element-view-emits'
import type { EditDeviceStructElementViewProps } from './edit-device-struct-element-view-props'
import { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data';

const props = defineProps<EditDeviceStructElementViewProps>()
const emits = defineEmits<EditDeviceStructElementViewEmits>()

const check_when_inited: Ref<boolean> = ref(props.struct_obj.check_when_inited)

async function apply(): Promise<void> {
    const device_struct = new DeviceStructElementData()
    device_struct.id = props.struct_obj.id
    device_struct.device_name = props.struct_obj.device_name
    device_struct.check_when_inited = check_when_inited.value
    emits('requested_update_device_struct', device_struct)
}
</script>
