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
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-close</v-icon>
                </v-btn>
            </div>

            <div class="gkill-floating-dialog__body">
                <ConfirmDeleteSkillView :application_config="application_config" :gkill_api="gkill_api"
                    :skill="skill"
                    @requested_delete_skill="(target: SkillInfo) => emits('requested_delete_skill', target)"
                    @requested_close_dialog="hide()" />
            </div>
        </div>
    </Teleport>
</template>
<script lang="ts" setup>
import type { ConfirmDeleteSkillDialogEmits } from './confirm-delete-skill-dialog-emits'
import type { ConfirmDeleteSkillDialogProps } from './confirm-delete-skill-dialog-props'
import ConfirmDeleteSkillView from '../views/confirm-delete-skill-view.vue'
import type { SkillInfo } from '@/classes/api/req_res/get-skill-list-response'
import { i18n } from '@/i18n'
import { useConfirmDeleteSkillDialog } from '@/classes/use-confirm-delete-skill-dialog'

const props = defineProps<ConfirmDeleteSkillDialogProps>()
const emits = defineEmits<ConfirmDeleteSkillDialogEmits>()
const { is_show_dialog, ui, skill, show, hide } = useConfirmDeleteSkillDialog({ props, emits })
defineExpose({ show, hide })
</script>
