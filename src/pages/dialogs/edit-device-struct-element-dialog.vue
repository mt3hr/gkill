<template>
    <EditDeviceStructElementView />
</template>
<script lang="ts" setup>
import type { EditDeviceStructElementDialogEmits } from './edit-device-struct-element-dialog-emits';
import type { EditDeviceStructElementDialogProps } from './edit-device-struct-element-dialog-props';
import EditDeviceStructElementView from '../views/edit-device-struct-element-view.vue';

const props = defineProps<EditDeviceStructElementDialogProps>();
const emits = defineEmits<EditDeviceStructElementDialogEmits>();
</script>