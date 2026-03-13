<template>
    <v-card class="pa-2">
        <v-text-field v-model="title" :label="i18n.global.t('DNOTE_TITLE_TITLE')" />
        <v-text-field v-model="prefix" :label="i18n.global.t('DNOTE_PREFIX_TITLE')" />
        <v-text-field v-model="suffix" :label="i18n.global.t('DNOTE_SUFFIX_TITLE')" />
        <v-select v-model="aggregate_target" :items="aggregate_targets" item-title="label" item-value="value"
            :label="i18n.global.t('DNOTE_AGGREGATE_TARGET')"></v-select>
        <v-select v-model="key_getter" :items="key_getters" item-title="label" item-value="value"
            :label="i18n.global.t('DNOTE_LIST_KEY')"></v-select>
        <PredicateGroup v-model="root_predicate" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()">{{ i18n.global.t("RESET_TITLE") }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()">{{ i18n.global.t("SAVE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import PredicateGroup from './edit-dnote-predicate-group.vue'
import type AddDnoteListViewEmits from './add-dnote-list-view-emits'
import type AddDnoteListViewProps from './add-dnote-list-view-props'
import { useAddDnoteListView } from '@/classes/use-add-dnote-list-view'

const props = defineProps<AddDnoteListViewProps>()
const emits = defineEmits<AddDnoteListViewEmits>()

const {
    title,
    prefix,
    suffix,
    root_predicate,
    aggregate_targets,
    aggregate_target,
    key_getters,
    key_getter,
    reset,
    save,
} = useAddDnoteListView({ props, emits })
</script>
