<template>
    <Teleport to="body" v-if="is_show_dialog">
        <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

        <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
            :class="ui.isTransparent.value ? 'is-transparent' : ''">
            <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
                @touchstart="ui.onHeaderPointerDown">
                <div class="gkill-floating-dialog__title">{{ skill?.name ?? "" }}</div>
                <div class="gkill-floating-dialog__spacer"></div>
                <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
                    :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-close</v-icon>
                </v-btn>
            </div>

            <div class="gkill-floating-dialog__body">
                <v-card variant="flat" class="pa-2">
                    <v-progress-linear v-if="is_loading" indeterminate color="primary" />
                    <div v-if="skill !== null && skill.invalid_reason !== ''" class="pa-2 browse-skill-invalid">
                        {{ i18n.global.t("SKILL_INVALID_TITLE") }}: {{ skill.invalid_reason }}
                    </div>
                    <div v-if="skill !== null" class="browse-skill-layout">
                        <v-list density="compact" class="browse-skill-file-list" :aria-label="i18n.global.t('FILE_TITLE')">
                            <v-list-item v-for="file in skill.files" :key="file.path"
                                :active="file.path === selected_path" @click="select_file(file)">
                                <v-list-item-title class="browse-skill-file-path">{{ file.path }}</v-list-item-title>
                                <v-list-item-subtitle>{{ file.size }} B</v-list-item-subtitle>
                            </v-list-item>
                        </v-list>
                        <div class="browse-skill-content">
                            <div v-if="is_binary" class="pa-2">{{ i18n.global.t("SKILL_BINARY_FILE_MESSAGE") }}</div>
                            <!-- 素の文字として出す（Markdown / HTML として描かない。v-html を使わない） -->
                            <pre v-else class="browse-skill-text">{{ text }}</pre>
                        </div>
                    </div>
                </v-card>
            </div>
        </div>
    </Teleport>
</template>
<script lang="ts" setup>
import type { BrowseSkillFilesDialogEmits } from './browse-skill-files-dialog-emits'
import type { BrowseSkillFilesDialogProps } from './browse-skill-files-dialog-props'
import { i18n } from '@/i18n'
import { useBrowseSkillFilesDialog } from '@/classes/use-browse-skill-files-dialog'

const props = defineProps<BrowseSkillFilesDialogProps>()
const emits = defineEmits<BrowseSkillFilesDialogEmits>()
const { is_show_dialog, ui, skill, selected_path, text, is_binary, is_loading, show, hide, select_file } = useBrowseSkillFilesDialog({ props, emits })
defineExpose({ show, hide })
</script>
<style lang="css" scoped>
.browse-skill-layout {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}

.browse-skill-file-list {
    flex: 0 0 auto;
    min-width: 200px;
    max-height: 60vh;
    overflow-y: auto;
}

.browse-skill-file-path {
    font-family: 'Consolas', 'Courier New', monospace;
}

.browse-skill-content {
    flex: 1 1 360px;
    min-width: 0;
}

.browse-skill-text {
    max-height: 60vh;
    overflow: auto;
    padding: 8px;
    font-family: 'Consolas', 'Courier New', monospace;
    font-size: 13px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-all;
    margin: 0;
}

.browse-skill-invalid {
    color: rgb(var(--v-theme-error));
}
</style>
