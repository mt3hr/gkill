<template>
    <v-card class="pa-2">
        <v-text-field v-model="title" :label="i18n.global.t('DNOTE_TITLE_TITLE')" />
        <v-text-field v-model="prefix" :label="i18n.global.t('DNOTE_PREFIX_TITLE')" />
        <v-text-field v-model="suffix" :label="i18n.global.t('DNOTE_SUFFIX_TITLE')" />
        <v-select v-model="aggregate_target" :items="aggregate_targets" item-title="label" item-value="value"
            :label="i18n.global.t('DNOTE_AGGREGATE_TARGET')"></v-select>
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
import type EditDnoteItemViewEmits from './edit-dnote-item-view-emits'
import type EditDnoteItemViewProps from './edit-dnote-item-view-props'
import DnoteItem from '@/classes/dnote/dnote-item'
import { useEditDnoteItemView } from '@/classes/use-edit-dnote-item-view'

const props = defineProps<EditDnoteItemViewProps>()
const emits = defineEmits<EditDnoteItemViewEmits>()
const model_value = defineModel<DnoteItem>()

const {
    // State
    title,
    prefix,
    suffix,
    root_predicate,
    aggregate_targets,
    aggregate_target,

    // Business logic
    reset,
    save,
} = useEditDnoteItemView({ props, emits, model_value })
</script>
