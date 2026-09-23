<template>
    <v-card variant="flat" class="pa-2">
        <v-card-title>
            {{ i18n.global.t("DELETE_SKILL_TITLE") }}
        </v-card-title>
        <div>{{ i18n.global.t("CONFIRM_DELETE_MESSAGE") }}</div>
        <div class="confirm-delete-skill-name">{{ skill.name }}</div>
        <div>{{ i18n.global.t("DELETE_SKILL_WARNING_MESSAGE") }}</div>
        <v-row class="pa-0 ma-0 gkill-dialog-actions">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary"
                    @click="emits('requested_delete_skill', skill); emits('requested_close_dialog')">{{
                        i18n.global.t("DELETE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ConfirmDeleteSkillViewEmits } from './confirm-delete-skill-view-emits'
import type { ConfirmDeleteSkillViewProps } from './confirm-delete-skill-view-props'

defineProps<ConfirmDeleteSkillViewProps>()
const emits = defineEmits<ConfirmDeleteSkillViewEmits>()
</script>
<style lang="css" scoped>
.confirm-delete-skill-name {
    font-family: 'Consolas', 'Courier New', monospace;
    padding: 4px 0;
}
</style>
