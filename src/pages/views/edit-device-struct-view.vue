<template>
    <DeviceStructElement :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
        :is_open="true" :struct_obj="application_config.device_struct" />
</template>
<script setup lang="ts">
import type { EditDeviceStructViewEmits } from './edit-device-struct-view-emits'
import type { EditDeviceStructViewProps } from './edit-device-struct-view-props'
import TagStructElement from './tag-struct-element.vue'
import DeviceStructElement from './device-struct-element.vue'
import AddNewDeviceStructElementDialog from '../dialogs/add-new-device-struct-element-dialog.vue'
import EditDeviceStructElementDialog from '../dialogs/edit-device-struct-element-dialog.vue'
import { ref } from 'vue'

const props = defineProps<EditDeviceStructViewProps>()
const emits = defineEmits<EditDeviceStructViewEmits>()
</script>
