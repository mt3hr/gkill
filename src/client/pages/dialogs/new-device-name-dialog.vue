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
          <v-card-title>{{ i18n.global.t('ADD_DEVICE_TITLE') }}</v-card-title>
          <v-row class="ma-0 pa-0">
            <v-col cols="13" class="ma-0 pa-0">
              <v-text-field v-model="device_name" :label="i18n.global.t('DEVICE_NAME_TITLE')" />
            </v-col>
          </v-row>
          <v-row class="ma-0 pa-0">
            <v-spacer />
            <v-col cols="auto" class="ma-0 pa-0">
              <v-btn color="primary" @click="emits_board_name" dark>{{ i18n.global.t('ADD_DEVICE_TITLE')
              }}</v-btn>
            </v-col>
          </v-row>
        </v-card>
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import type { NewDeviceNameDialogEmits } from './new-device-name-dialog-emits';
import type { NewDeviceNameDialogProps } from './new-device-name-dialog-props';
import { useNewDeviceNameDialog } from '@/classes/use-new-device-name-dialog'
import { i18n } from '@/i18n'

const props = defineProps<NewDeviceNameDialogProps>()
const emits = defineEmits<NewDeviceNameDialogEmits>()

const { is_show_dialog, ui, device_name, show, hide, emits_board_name } = useNewDeviceNameDialog({ props, emits })

defineExpose({ show, hide })
</script>

