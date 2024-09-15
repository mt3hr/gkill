<template>
    <ApplicationConfigView />
</template>

<script lang="ts" setup>
import { defineProps } from 'vue';
import type { ApplicationConfigDialogProps } from './application-config-dialog-props';
import type { ApplicationConfigDialogEmits } from './application-config-dialog-emits';
import ApplicationConfigView from '../views/application-config-view.vue';


const props = defineProps<ApplicationConfigDialogProps>();
const emits = defineEmits<ApplicationConfigDialogEmits>();
</script>
