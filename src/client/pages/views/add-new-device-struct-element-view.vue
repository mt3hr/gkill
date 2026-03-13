<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("ADD_DEVICE_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="device_name" :label="i18n.global.t('DEVICE_TITLE')" />
        <v-checkbox v-model="check_when_inited" hide-detail :label="i18n.global.t('CHECK_WHEN_INITED_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="emits_device_name">{{ i18n.global.t("ADD_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">
                        {{ i18n.global.t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AddNewDeviceStructElementViewEmits } from './add-new-device-struct-element-view-emits'
import type { AddNewDeviceStructElementViewProps } from './add-new-device-struct-element-view-props'
import { useAddNewDeviceStructElementView } from '@/classes/use-add-new-device-struct-element-view'

const props = defineProps<AddNewDeviceStructElementViewProps>()
const emits = defineEmits<AddNewDeviceStructElementViewEmits>()

const {
    device_name,
    check_when_inited,
    emits_device_name,
    reset_device_name,
} = useAddNewDeviceStructElementView({ props, emits })

defineExpose({ reset_device_name })
</script>
