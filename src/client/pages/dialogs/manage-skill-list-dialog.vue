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
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="help_dialog?.show()" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-help-circle-outline</v-icon>
                </v-btn>
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-close</v-icon>
                </v-btn>
            </div>

            <div class="gkill-floating-dialog__body">
                <v-card variant="flat" class="pa-2">
                    <ManageSkillListView :application_config="application_config" :gkill_api="gkill_api"
                        :skills="skills" :is_loading="is_loading" :is_uploading="is_uploading"
                        @requested_browse_skill="(skill: SkillInfo) => show_browse_skill_files_dialog(skill)"
                        @requested_download_skill="(skill: SkillInfo) => download_skill(skill)"
                        @requested_show_confirm_delete_skill_dialog="(skill: SkillInfo) => show_confirm_delete_skill_dialog(skill)"
                        @selected_upload_file="(e: Event) => onSelectedUploadFile(e)" />
                    <BrowseSkillFilesDialog :application_config="application_config" :gkill_api="gkill_api"
                        @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
                        @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
                        ref="browse_skill_files_dialog" />
                    <ConfirmUploadSkillDialog :application_config="application_config" :gkill_api="gkill_api"
                        @requested_apply_upload_skill="() => apply_upload_skill()"
                        @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
                        @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
                        ref="confirm_upload_skill_dialog" />
                    <ConfirmDeleteSkillDialog :application_config="application_config" :gkill_api="gkill_api"
                        @requested_delete_skill="(skill: SkillInfo) => delete_skill(skill)"
                        @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
                        @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
                        ref="confirm_delete_skill_dialog" />
                </v-card>
                <HelpDialog screen_name="mcp" ref="help_dialog" />
            </div>
        </div>
    </Teleport>
</template>
<script lang="ts" setup>
import type { ManageSkillListDialogEmits } from './manage-skill-list-dialog-emits'
import type { ManageSkillListDialogProps } from './manage-skill-list-dialog-props'
import ManageSkillListView from '../views/manage-skill-list-view.vue'
import BrowseSkillFilesDialog from './browse-skill-files-dialog.vue'
import ConfirmUploadSkillDialog from './confirm-upload-skill-dialog.vue'
import ConfirmDeleteSkillDialog from './confirm-delete-skill-dialog.vue'
import HelpDialog from './help-dialog.vue'
import type { SkillInfo } from '@/classes/api/req_res/get-skill-list-response'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import { useManageSkillListDialog } from '@/classes/use-manage-skill-list-dialog'

const props = defineProps<ManageSkillListDialogProps>()
const emits = defineEmits<ManageSkillListDialogEmits>()
const {
    browse_skill_files_dialog,
    confirm_upload_skill_dialog,
    confirm_delete_skill_dialog,
    help_dialog,
    skills,
    is_loading,
    is_uploading,
    is_show_dialog,
    ui,
    show,
    hide,
    show_browse_skill_files_dialog,
    show_confirm_delete_skill_dialog,
    download_skill,
    onSelectedUploadFile,
    apply_upload_skill,
    delete_skill,
} = useManageSkillListDialog({ props, emits })
defineExpose({ show, hide })
</script>
