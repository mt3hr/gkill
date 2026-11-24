<template>
    <v-card variant="flat" class="pa-2 export_kyou_html_view">
        <v-text-field v-model="title" :label="i18n.global.t('SHARE_KYOU_TITLE_TITLE')" density="compact" hide-details
            class="mb-3 export_kyou_html_title_field" />

        <div class="text-subtitle-2">{{ i18n.global.t('EXPORT_TARGET_TITLE') }}</div>
        <v-radio-group v-model="target" density="compact" hide-details class="mb-2">
            <v-radio :label="i18n.global.t('EXPORT_TARGET_ALL_COLUMNS')" value="all"
                class="export_kyou_html_target_all" />
            <v-radio :label="i18n.global.t('EXPORT_TARGET_FOCUSED_COLUMN')" value="focused"
                :disabled="!has_focused_column" class="export_kyou_html_target_focused" />
        </v-radio-group>

        <div class="text-subtitle-2">{{ i18n.global.t('EXPORT_INCLUDE_TITLE') }}</div>
        <v-row no-gutters class="mb-2">
            <v-col cols="6">
                <v-checkbox v-model="include_tags" :label="i18n.global.t('TAG_TITLE')" density="compact" hide-details />
                <v-checkbox v-model="include_notifications" :label="i18n.global.t('NOTIFICATION_TITLE')"
                    density="compact" hide-details />
                <v-checkbox v-model="include_gps_log" :label="i18n.global.t('EXPORT_INCLUDE_GPS_LOG')" density="compact"
                    hide-details />
            </v-col>
            <v-col cols="6">
                <v-checkbox v-model="include_texts" :label="i18n.global.t('TEXT_TITLE')" density="compact"
                    hide-details />
                <v-checkbox v-model="include_timeiss" :label="i18n.global.t('EXPORT_INCLUDE_TIMEISS')" density="compact"
                    hide-details />
                <v-checkbox v-model="include_plugin_content" :label="i18n.global.t('EXPORT_INCLUDE_PLUGIN_CONTENT')"
                    density="compact" hide-details />
            </v-col>
        </v-row>

        <div class="text-subtitle-2">{{ i18n.global.t('EXPORT_SCREEN_TITLE') }}</div>
        <v-row no-gutters class="mb-2">
            <v-col cols="6">
                <v-checkbox v-model="include_dnote" :label="i18n.global.t('EXPORT_SCREEN_DNOTE')" density="compact"
                    hide-details />
                <v-checkbox v-model="include_sidebar" :label="i18n.global.t('EXPORT_SCREEN_SIDEBAR')" density="compact"
                    hide-details />
            </v-col>
            <v-col cols="6">
                <v-checkbox v-model="include_map" :label="i18n.global.t('EXPORT_SCREEN_MAP')" density="compact"
                    hide-details />
            </v-col>
        </v-row>

        <div class="text-subtitle-2">{{ i18n.global.t('EXPORT_MEDIA_TITLE') }}</div>
        <v-radio-group v-model="media_mode" density="compact" hide-details class="mb-2">
            <v-radio :label="i18n.global.t('EXPORT_MEDIA_THUMBNAIL')" value="thumbnail" />
            <v-radio :label="i18n.global.t('EXPORT_MEDIA_FULL')" value="full" />
            <v-radio :label="i18n.global.t('EXPORT_MEDIA_NONE')" value="none"
                class="export_kyou_html_media_none" />
        </v-radio-group>

        <v-row class="pa-0 ma-0 mb-2" dense>
            <v-col cols="6">
                <v-text-field v-model.number="media_item_limit_mb" type="number" min="1" density="compact"
                    hide-details suffix="MB" :disabled="media_mode === 'none'"
                    :label="i18n.global.t('EXPORT_MEDIA_ITEM_LIMIT_TITLE')"
                    class="export_kyou_html_media_item_limit_field" />
            </v-col>
            <v-col cols="6">
                <v-text-field v-model.number="media_budget_mb" type="number" min="1" density="compact"
                    hide-details suffix="MB" :disabled="media_mode === 'none'"
                    :label="i18n.global.t('EXPORT_MEDIA_BUDGET_TITLE')"
                    class="export_kyou_html_media_budget_field" />
            </v-col>
        </v-row>

        <div class="text-subtitle-2">{{ i18n.global.t('EXPORT_FORMAT_TITLE') }}</div>
        <v-radio-group v-model="output_format" density="compact" hide-details class="mb-2">
            <v-radio :label="i18n.global.t('EXPORT_FORMAT_SINGLE')" value="single" />
            <v-radio :label="i18n.global.t('EXPORT_FORMAT_SIDECAR')" value="sidecar"
                :disabled="!is_sidecar_selectable" class="export_kyou_html_format_sidecar" />
        </v-radio-group>

        <v-card-actions class="pa-0">
            <v-spacer />
            <!-- 出力中はキャンセルが中断になる（押せないままだと止める手段が無い） -->
            <v-btn color="secondary" variant="flat" @click="cancel()">
                {{ i18n.global.t('CANCEL_TITLE') }}
            </v-btn>
            <v-btn color="primary" variant="flat" :loading="is_exporting" class="export_kyou_html_execute_button"
                @click="export_html()">
                {{ i18n.global.t('EXPORT_EXECUTE_TITLE') }}
            </v-btn>
        </v-card-actions>
    </v-card>
</template>

<script lang="ts" setup>
import type { ExportKyouHTMLViewEmits } from './export-kyou-html-view-emits'
import type { ExportKyouHTMLViewProps } from './export-kyou-html-view-props'
import { i18n } from '@/i18n'
import { useExportKyouHTMLView } from '@/classes/use-export-kyou-html-view'

const props = defineProps<ExportKyouHTMLViewProps>()
const emits = defineEmits<ExportKyouHTMLViewEmits>()
const {
    title,
    target,
    include_tags,
    include_texts,
    include_notifications,
    include_timeiss,
    include_gps_log,
    include_plugin_content,
    include_dnote,
    include_map,
    include_sidebar,
    media_mode,
    media_budget_mb,
    media_item_limit_mb,
    output_format,
    is_exporting,
    has_focused_column,
    is_sidecar_selectable,
    export_html,
    cancel,
} = useExportKyouHTMLView({ props, emits })
</script>

<style lang="css" scoped>
.export_kyou_html_view {
    min-width: 320px;
}
</style>
