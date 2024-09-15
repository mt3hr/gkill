<template>
    <EditDeviceStructView />
</template>
<script lang="ts" setup>
import EditDeviceStructView from '../views/edit-device-struct-view.vue';
import type { EditDeviceStructgDialogEmits } from './edit-device-structg-dialog-emits';
import type { EditDeviceStructgDialogProps } from './edit-device-structg-dialog-props';

const props = defineProps<EditDeviceStructgDialogProps>();
const emits = defineEmits<EditDeviceStructgDialogEmits>();
</script>