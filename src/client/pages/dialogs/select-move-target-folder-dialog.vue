<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card variant="flat" class="pa-2">
          <v-card-title>{{ i18n.global.t("SELECT_MOVE_TARGET_FOLDER_TITLE") }}</v-card-title>
          <v-list>
            <v-list-item @click="select_folder(null)">
              <v-list-item-title>{{ i18n.global.t("MOVE_TARGET_ROOT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item v-for="candidate in folder_candidates" :key="candidate.id ? candidate.id : ''"
              @click="select_folder(candidate.id)">
              <v-list-item-title :style="{ paddingLeft: ((candidate.depth + 1) * 16) + 'px' }">
                {{ candidate.name }}
              </v-list-item-title>
            </v-list-item>
          </v-list>
          <v-card-actions class="gkill-dialog-actions">
            <v-btn color="secondary" @click="hide">{{ i18n.global.t("CANCEL_TITLE") }}</v-btn>
          </v-card-actions>
        </v-card>
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import type { SelectMoveTargetFolderDialogEmits } from './select-move-target-folder-dialog-emits'
import type { SelectMoveTargetFolderDialogProps } from './select-move-target-folder-dialog-props'
import type { FoldableStructModel } from '../views/foldable-struct-model'
import { list_move_target_folders, type MoveTargetFolderCandidate } from '@/classes/foldable-struct-move'
import { closeDialogViaHistory, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"
import { i18n } from '@/i18n'

defineProps<SelectMoveTargetFolderDialogProps>()
const emits = defineEmits<SelectMoveTargetFolderDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)
const moving_struct_id: Ref<string> = ref("")
const folder_candidates: Ref<Array<MoveTargetFolderCandidate>> = ref([])

useDialogHistoryStack(is_show_dialog)
const ui = useFloatingDialog("select-move-target-folder-dialog", {
  centerMode: "always",
  onEscape: () => hide(),
})

async function show(root_struct_obj: FoldableStructModel, struct_id: string): Promise<void> {
  moving_struct_id.value = struct_id
  folder_candidates.value = list_move_target_folders(root_struct_obj, struct_id)
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  closeDialogViaHistory(is_show_dialog)
}
function select_folder(target_folder_id: string | null): void {
  emits('requested_move_struct_obj_to_folder', moving_struct_id.value, target_folder_id)
  hide()
}
</script>
