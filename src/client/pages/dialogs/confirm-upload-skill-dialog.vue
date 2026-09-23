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
                <ConfirmUploadSkillView :application_config="application_config" :gkill_api="gkill_api"
                    :plan="plan"
                    @requested_apply_upload_skill="(replace_plan: SkillReplacePlan) => emits('requested_apply_upload_skill', replace_plan)"
                    @requested_close_dialog="hide()" />
            </div>
        </div>
    </Teleport>
</template>
<script lang="ts" setup>
import type { ConfirmUploadSkillDialogEmits } from './confirm-upload-skill-dialog-emits'
import type { ConfirmUploadSkillDialogProps } from './confirm-upload-skill-dialog-props'
import ConfirmUploadSkillView from '../views/confirm-upload-skill-view.vue'
import type { SkillReplacePlan } from '@/classes/api/req_res/upload-skill-response'
import { i18n } from '@/i18n'
import { useConfirmUploadSkillDialog } from '@/classes/use-confirm-upload-skill-dialog'

const props = defineProps<ConfirmUploadSkillDialogProps>()
const emits = defineEmits<ConfirmUploadSkillDialogEmits>()
const { is_show_dialog, ui, plan, show, hide } = useConfirmUploadSkillDialog({ props, emits })
defineExpose({ show, hide })
</script>
