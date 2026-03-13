<template>
  <Teleport to="body" v-if="is_show_dialog" >
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
  <v-checkbox v-model="ui.isTransparent.value" color="white"    size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'" variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">

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
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import type { NewBoardNameDialogEmits } from './new-board-name-dialog-emits'
import type { NewBoardNameDialogProps } from './new-board-name-dialog-props'
import { useNewBoardNameDialog } from '@/classes/use-new-board-name-dialog'
import { i18n } from '@/i18n'

const props = defineProps<NewBoardNameDialogProps>()
const emits = defineEmits<NewBoardNameDialogEmits>()

const { is_show_dialog, ui, board_name, show, hide, emits_board_name } = useNewBoardNameDialog({ props, emits })

defineExpose({ show, hide })
</script>

