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
import type { SelectMoveTargetFolderDialogEmits } from './select-move-target-folder-dialog-emits'
import type { SelectMoveTargetFolderDialogProps } from './select-move-target-folder-dialog-props'
import { i18n } from '@/i18n'
import { useSelectMoveTargetFolderDialog } from '@/classes/use-select-move-target-folder-dialog'

const props = defineProps<SelectMoveTargetFolderDialogProps>()
const emits = defineEmits<SelectMoveTargetFolderDialogEmits>()
const { is_show_dialog, folder_candidates, ui, show, hide, select_folder } = useSelectMoveTargetFolderDialog({ props, emits })
defineExpose({ show, hide })
</script>
