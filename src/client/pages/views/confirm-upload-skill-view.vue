<template>
    <v-card variant="flat" class="pa-2">
        <v-card-title>
            {{ i18n.global.t("CONFIRM_UPLOAD_SKILL_TITLE") }}: {{ plan.name }}
        </v-card-title>
        <div class="pa-2">
            {{ plan.is_new ? i18n.global.t("CONFIRM_NEW_SKILL_MESSAGE") : i18n.global.t("CONFIRM_REPLACE_SKILL_MESSAGE") }}
        </div>
        <div v-if="!plan.is_new && plan.added.length === 0 && plan.removed.length === 0 && plan.changed.length === 0"
            class="pa-2">
            {{ i18n.global.t("SKILL_NO_CHANGES_MESSAGE") }}
        </div>
        <div v-if="plan.added.length !== 0" class="pa-2 confirm-upload-skill-added">
            <div class="font-weight-bold">{{ i18n.global.t("SKILL_ADDED_FILES_TITLE") }}</div>
            <div v-for="path in plan.added" :key="path" class="confirm-upload-skill-path">{{ path }}</div>
        </div>
        <div v-if="plan.changed.length !== 0" class="pa-2 confirm-upload-skill-changed">
            <div class="font-weight-bold">{{ i18n.global.t("SKILL_CHANGED_FILES_TITLE") }}</div>
            <div v-for="path in plan.changed" :key="path" class="confirm-upload-skill-path">{{ path }}</div>
        </div>
        <div v-if="plan.removed.length !== 0" class="pa-2 confirm-upload-skill-removed">
            <div class="font-weight-bold">{{ i18n.global.t("SKILL_REMOVED_FILES_TITLE") }}</div>
            <div v-for="path in plan.removed" :key="path" class="confirm-upload-skill-path">{{ path }}</div>
        </div>
        <div v-if="plan.ignored.length !== 0" class="pa-2 confirm-upload-skill-ignored">
            <div class="font-weight-bold">{{ i18n.global.t("SKILL_IGNORED_FILES_TITLE") }}</div>
            <div v-for="path in plan.ignored" :key="path" class="confirm-upload-skill-path">{{ path }}</div>
        </div>
        <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary"
                    @click="emits('requested_apply_upload_skill', plan); emits('requested_close_dialog')">{{
                        i18n.global.t("APPLY_TITLE") }}</v-btn>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{
                    i18n.global.t("CANCEL_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ConfirmUploadSkillViewEmits } from './confirm-upload-skill-view-emits'
import type { ConfirmUploadSkillViewProps } from './confirm-upload-skill-view-props'

defineProps<ConfirmUploadSkillViewProps>()
const emits = defineEmits<ConfirmUploadSkillViewEmits>()
</script>
<style lang="css" scoped>
.confirm-upload-skill-path {
    font-family: 'Consolas', 'Courier New', monospace;
    padding-left: 12px;
    word-break: break-all;
}

.confirm-upload-skill-removed .confirm-upload-skill-path {
    color: rgb(var(--v-theme-error));
}
</style>
