<template>
    <AddRepDialog />
    <ConfirmDeleteRepDialog />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import type { AllocateRepViewEmits } from './allocate-rep-view-emits';
import type { AllocateRepViewProps } from './allocate-rep-view-props';
import { ServerConfig } from '@/classes/datas/config/server-config';
import AddRepDialog from '../dialogs/add-rep-dialog.vue';
import ConfirmDeleteRepDialog from '../dialogs/confirm-delete-rep-dialog.vue';

const props = defineProps<AllocateRepViewProps>();
const emits = defineEmits<AllocateRepViewEmits>();
const cloned_server_config: Ref<ServerConfig> = ref(await props.server_config.clone());
</script>