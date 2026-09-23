<template>
    <v-card variant="flat">
        <v-card-title>
            {{ i18n.global.t("SKILL_LIST_TITLE") }}
        </v-card-title>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <!-- label で包んだ隠し input。ボタンを押すとファイル選択が開く（script 無しで済ませるため） -->
                <v-btn tag="label" dark color="primary" :loading="is_uploading" :disabled="is_uploading"
                    class="manage-skill-upload-button">
                    {{ i18n.global.t("SKILL_UPLOAD_TITLE") }}
                    <input type="file" accept=".zip,application/zip" hidden
                        @change="(e: Event) => emits('selected_upload_file', e)" />
                </v-btn>
            </v-col>
        </v-row>
        <v-progress-linear v-if="is_loading" indeterminate color="primary" />
        <div v-if="!is_loading && skills.length === 0" class="pa-2">
            {{ i18n.global.t("SKILL_EMPTY_MESSAGE") }}
        </div>
        <v-table v-if="skills.length !== 0" density="compact">
            <thead>
                <tr>
                    <th scope="col">{{ i18n.global.t("SKILL_NAME_TITLE") }}</th>
                    <th scope="col">{{ i18n.global.t("DESCRIPTION_TITLE") }}</th>
                    <th scope="col">{{ i18n.global.t("SKILL_UPDATED_TIME_TITLE") }}</th>
                    <th scope="col">{{ i18n.global.t("SKILL_FILE_COUNT_TITLE") }}</th>
                    <th scope="col"></th>
                </tr>
            </thead>
            <tbody>
                <tr v-for="skill in skills" :key="skill.name" class="manage-skill-row">
                    <td class="manage-skill-name">{{ skill.name }}</td>
                    <td>
                        <div v-if="skill.invalid_reason !== ''" class="manage-skill-invalid">
                            {{ i18n.global.t("SKILL_INVALID_TITLE") }}: {{ skill.invalid_reason }}
                        </div>
                        <div v-else>{{ skill.description }}</div>
                    </td>
                    <td class="manage-skill-nowrap">{{ skill.updated_time ? format_time(new Date(skill.updated_time)) : "" }}</td>
                    <td class="manage-skill-nowrap">{{ skill.file_count }}</td>
                    <td class="manage-skill-nowrap">
                        <v-btn color="primary" size="small" @click="emits('requested_browse_skill', skill)">
                            {{ i18n.global.t("SKILL_VIEW_TITLE") }}
                        </v-btn>
                        <v-btn color="primary" size="small" @click="emits('requested_download_skill', skill)">
                            {{ i18n.global.t("ZIP_DOWNLOAD_LINK_TITLE") }}
                        </v-btn>
                        <v-btn dark color="secondary" size="small"
                            @click="emits('requested_show_confirm_delete_skill_dialog', skill)">
                            {{ i18n.global.t("DELETE_TITLE") }}
                        </v-btn>
                    </td>
                </tr>
            </tbody>
        </v-table>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { format_time } from '@/classes/format-date-time'
import type { ManageSkillListViewEmits } from './manage-skill-list-view-emits'
import type { ManageSkillListViewProps } from './manage-skill-list-view-props'

defineProps<ManageSkillListViewProps>()
const emits = defineEmits<ManageSkillListViewEmits>()
</script>
<style lang="css" scoped>
.manage-skill-nowrap {
    white-space: nowrap;
}

.manage-skill-name {
    font-family: 'Consolas', 'Courier New', monospace;
    white-space: nowrap;
}

.manage-skill-invalid {
    color: rgb(var(--v-theme-error));
}
</style>
