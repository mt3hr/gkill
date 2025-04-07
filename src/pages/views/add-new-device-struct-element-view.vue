<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ $t("ADD_DEVICE_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="device_name" :label="$t('DEVICE_TITLE')" />
        <v-checkbox v-model="check_when_inited" hide-detail :label="$t('CHECK_WHEN_INITED_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="emits_device_name">{{ $t("ADD_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">
                        {{ $t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data';
import { type Ref, ref } from 'vue';
import type { AddNewDeviceStructElementViewEmits } from './add-new-device-struct-element-view-emits'
import type { AddNewDeviceStructElementViewProps } from './add-new-device-struct-element-view-props'
import { GkillError } from '@/classes/api/gkill-error';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<AddNewDeviceStructElementViewProps>()
const emits = defineEmits<AddNewDeviceStructElementViewEmits>()

defineExpose({ reset_device_name })

const device_name: Ref<string> = ref("")
const check_when_inited: Ref<boolean> = ref(true)
const is_force_hide: Ref<boolean> = ref(false)

function emits_device_name(): void {
    if (device_name.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.device_name_is_blank
        error.error_message = t("DEVICE_NAME_IS_BLANK_MESSAGE")
        emits('received_errors', [error])
        return
    }

    const device_struct_element = new DeviceStructElementData()
    device_struct_element.id = props.gkill_api.generate_uuid()
    device_struct_element.check_when_inited = check_when_inited.value
    device_struct_element.children = null
    device_struct_element.indeterminate = false
    device_struct_element.key = device_name.value
    device_struct_element.device_name = device_name.value
    emits('requested_add_device_struct_element', device_struct_element)
    emits('requested_close_dialog')
}

function reset_device_name(): void {
    device_name.value = ""
    check_when_inited.value = true
    is_force_hide.value = false
}
</script>
