<template>
    <v-card class="pa-2" v-if="id !== ''">
        <v-text-field v-model="title" :label="i18n.global.t('RYUU_TITLE_TITLE')" />
        <v-text-field v-model="prefix" :label="i18n.global.t('RYUU_PREFIX_TITLE')" />
        <v-text-field v-model="suffix" :label="i18n.global.t('RYUU_SUFFIX_TITLE')" />
        <PredicateGroup v-model="root_predicate" />
        <v-select v-model="related_time_match_type" :items="related_time_match_types" item-title="label"
            item-value="value" :label="i18n.global.t('RYUU_RELATION_PREDICATE_TITLE')"></v-select>
        <v-text-field type="number" v-model="find_duration_hour" min="0"
            :label="i18n.global.t('RYUU_FIND_DURATION_HOUR_TITLE')" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="is_use_custom_find_kyou_query"
                    :label="i18n.global.t('RYUU_CUSTOM_FIND_QUERY_TITLE')" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pt-3 ma-0">
                <v-btn v-if="is_use_custom_find_kyou_query" dark color="primary"
                    @click="show_find_query_editor_dialog()">{{
                        i18n.global.t('RYUU_FIND_QUERY_TITLE') }} </v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()">{{ i18n.global.t("RESET_TITLE") }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()">{{ i18n.global.t("SAVE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <FindQueryEditorDialog v-model="find_kyou_query" v-if="find_kyou_query" :application_config="application_config"
            :gkill_api="gkill_api" ref="find_query_editor_dialog" />
    </v-card>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import PredicateGroup from './edit-dnote-predicate-group.vue'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query'
import FindQueryEditorDialog from '../dialogs/find-query-editor-dialog.vue'
import type EditRyuuItemViewEmits from './edit-ryuu-item-view-emits'
import type EditRyuuItemViewProps from './edit-ryuu-item-view-props'
import { useEditRyuuItemView } from '@/classes/use-edit-ryuu-item-view'

const model_value = defineModel<RelatedKyouQuery>()
const props = defineProps<EditRyuuItemViewProps>()
const emits = defineEmits<EditRyuuItemViewEmits>()

const {
    // Template refs
    find_query_editor_dialog,

    // State
    id,
    title,
    prefix,
    suffix,
    related_time_match_type,
    find_kyou_query,
    find_duration_hour,
    is_use_custom_find_kyou_query,
    related_time_match_types,
    root_predicate,

    // Methods
    reset,
    save,
    show_find_query_editor_dialog,
} = useEditRyuuItemView({ props, emits, model_value })
</script>
