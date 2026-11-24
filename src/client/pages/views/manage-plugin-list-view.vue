<template>
    <v-card variant="flat">
        <v-card-title>
            {{ i18n.global.t("PLUGIN_CONFIG_TITLE") }}: {{ target_user_id }}
        </v-card-title>
        <v-progress-linear v-if="is_loading" indeterminate color="primary" />
        <div v-if="!is_loading && plugins.length === 0" class="pa-2">
            {{ i18n.global.t("PLUGIN_EMPTY_MESSAGE") }}
        </div>
        <v-table v-if="plugins.length !== 0" density="compact">
            <thead>
                <tr>
                    <th scope="col">{{ i18n.global.t("PLUGIN_NAME_TITLE") }}</th>
                    <th scope="col">{{ i18n.global.t("PLUGIN_VERSION_TITLE") }}</th>
                    <th scope="col">{{ i18n.global.t("PLUGIN_STATE_TITLE") }}</th>
                    <th scope="col"></th>
                </tr>
            </thead>
            <!-- 説明は行を分けて全幅に置く。ダイアログの本文は幅 min(500px, 85vw) で
                 overflow-x: hidden なので、説明を同じ行の列にすると「設定」ボタンが
                 枠の外へ押し出されて押せなくなる -->
            <tbody v-for="plugin in plugins" :key="plugin.name" class="manage-plugin-group">
                <tr>
                    <td class="manage-plugin-name">{{ plugin.name }}</td>
                    <td class="manage-plugin-nowrap">{{ plugin.version }}</td>
                    <td class="manage-plugin-nowrap">
                        <span v-if="plugin.is_alive">{{ i18n.global.t("PLUGIN_STATE_ALIVE_TITLE") }}</span>
                        <span v-else class="manage-plugin-dead">{{ i18n.global.t("PLUGIN_STATE_DEAD_TITLE") }}</span>
                    </td>
                    <td class="manage-plugin-nowrap">
                        <v-btn color="primary" size="small" @click="emits('requested_show_plugin_config', plugin)">
                            {{ i18n.global.t("TOOLTIP_SETTINGS") }}
                        </v-btn>
                    </td>
                </tr>
                <tr v-if="plugin.description">
                    <td colspan="4" class="manage-plugin-description">{{ plugin.description }}</td>
                </tr>
            </tbody>
        </v-table>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ManagePluginListViewEmits } from './manage-plugin-list-view-emits'
import type { ManagePluginListViewProps } from './manage-plugin-list-view-props'

defineProps<ManagePluginListViewProps>()
const emits = defineEmits<ManagePluginListViewEmits>()
</script>
<style lang="css" scoped>
.manage-plugin-nowrap {
    white-space: nowrap;
}

.manage-plugin-name {
    font-family: 'Consolas', 'Courier New', monospace;
    white-space: nowrap;
}

.manage-plugin-dead {
    color: rgb(var(--v-theme-error));
}

.manage-plugin-description {
    font-size: 0.8em;
    opacity: 0.8;
    padding-bottom: 8px;
}

/* プラグインの区切り。1プラグイン = tbody 1つ（名前の行と説明の行） */
.manage-plugin-group:not(:last-child) {
    border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
</style>
