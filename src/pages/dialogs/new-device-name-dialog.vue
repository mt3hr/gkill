<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <v-card>
            <v-card-title>{{ i18n.global.t('ADD_DEVICE_TITLE') }}</v-card-title>
            <v-row class="ma-0 pa-0">
                <v-col cols="13" class="ma-0 pa-0">
                    <v-text-field v-model="device_name" :label="i18n.global.t('DEVICE_NAME_TITLE')" />
                </v-col>
            </v-row>
            <v-row class="ma-0 pa-0">
                <v-spacer />
                <v-col cols="auto" class="ma-0 pa-0">
                    <v-btn color="primary" @click="emits_board_name" dark>{{ i18n.global.t('ADD_DEVICE_TITLE')
                    }}</v-btn>
                </v-col>
            </v-row>
        </v-card>
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { NewDeviceNameDialogEmits } from './new-device-name-dialog-emits';
import type { NewDeviceNameDialogProps } from './new-device-name-dialog-props';

defineProps<NewDeviceNameDialogProps>()
const emits = defineEmits<NewDeviceNameDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)
const device_name: Ref<string> = ref("")

async function show(): Promise<void> {
    device_name.value = ""
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
function emits_board_name(): void {
    emits('setted_new_device_name', device_name.value)
    hide()
}
</script>
