<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <v-card>
            <v-card-title>{{ i18n.global.t("ADD_MI_BOARD_TITLE") }}</v-card-title>
            <v-row class="ma-0 pa-0">
                <v-col cols="13" class="ma-0 pa-0">
                    <v-text-field v-model="board_name" :label="i18n.global.t('MI_BOARD_NAME_TITLE')" />
                </v-col>
            </v-row>
            <v-row class="ma-0 pa-0">
                <v-spacer />
                <v-col cols="auto" class="ma-0 pa-0">
                    <v-btn color="primary" @click="emits_board_name" dark>{{ i18n.global.t('ADD_MI_BOARD_TITLE')
                        }}</v-btn>
                </v-col>
            </v-row>
        </v-card>
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { NewBoardNameDialogEmits } from './new-board-name-dialog-emits'
import type { NewBoardNameDialogProps } from './new-board-name-dialog-props'

defineProps<NewBoardNameDialogProps>()
const emits = defineEmits<NewBoardNameDialogEmits>()
defineExpose({ show, hide })

import { useBackToCloseDialog } from '@/classes/use-back-to-close-dialog'
const is_show_dialog: Ref<boolean> = ref(false)
useBackToCloseDialog(is_show_dialog)
const board_name: Ref<string> = ref("")

async function show(): Promise<void> {
    board_name.value = ""
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
function emits_board_name(): void {
    emits('setted_new_board_name', board_name.value)
    hide()
}
</script>
