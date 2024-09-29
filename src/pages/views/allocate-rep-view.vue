<template>
    <!--//TODO 実装-->
    <AddRepDialog :application_config="application_config" :gkill_api="gkill_api" :server_config="server_config"
        @requested_add_rep="add_rep" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct" />
    <ConfirmDeleteRepDialog :application_config="application_config" :gkill_api="gkill_api"
        :rep_id="delete_target_rep ? delete_target_rep.id : ''" :server_config="server_config"
        @requested_delete_rep="(rep) => delete_rep(rep)" @requested_add_rep="add_rep"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import type { AllocateRepViewEmits } from './allocate-rep-view-emits'
import type { AllocateRepViewProps } from './allocate-rep-view-props'
import AddRepDialog from '../dialogs/add-rep-dialog.vue'
import ConfirmDeleteRepDialog from '../dialogs/confirm-delete-rep-dialog.vue'
import type { Repository } from '@/classes/datas/config/repository'

const props = defineProps<AllocateRepViewProps>()
const emits = defineEmits<AllocateRepViewEmits>()

const delete_target_rep: Ref<Repository | null> = ref(null)

async function add_rep(rep: Repository): Promise<void> {
    throw new Error('Not implemented')
}

async function delete_rep(rep: Repository): Promise<void> {
    throw new Error('Not implemented')
}
</script>
