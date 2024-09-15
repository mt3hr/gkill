<template>
    <AddNewDeviceStructElementView />
</template>
<script lang="ts" setup>
import type { AddNewDeviceStructElementDialogEmits } from './add-new-device-struct-element-dialog-emits';
import type { AddNewDeviceStructElementDialogProps } from './add-new-device-struct-element-dialog-props';
import AddNewDeviceStructElementView from '../views/add-new-device-struct-element-view.vue';

const props = defineProps<AddNewDeviceStructElementDialogProps>();
const emits = defineEmits<AddNewDeviceStructElementDialogEmits>();
</script>
