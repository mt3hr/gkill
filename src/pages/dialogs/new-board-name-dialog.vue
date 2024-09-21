<template>
    <v-dialog v-model="is_show_dialog">
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { NewBoardNameDialogEmits } from './new-board-name-dialog-emits'
import type { NewBoardNameDialogProps } from './new-board-name-dialog-props'

const props = defineProps<NewBoardNameDialogProps>()
const emits = defineEmits<NewBoardNameDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
