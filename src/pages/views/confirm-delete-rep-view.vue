<template>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ConfirmDeleteRepViewEmits } from './confirm-delete-rep-view-emits'
import type { ConfirmDeleteRepViewProps } from './confirm-delete-rep-view-props'
import { ServerConfig } from '@/classes/datas/config/server-config'

const props = defineProps<ConfirmDeleteRepViewProps>()
const emits = defineEmits<ConfirmDeleteRepViewEmits>()
const cloned_server_config: Ref<ServerConfig> = ref(await props.server_config.clone())
</script>
