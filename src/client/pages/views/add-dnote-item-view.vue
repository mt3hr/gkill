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
import type AddDnoteItemViewEmits from './add-dnote-item-view-emits'
import type AddDnoteItemViewProps from './add-dnote-item-view-props'
import { useAddDnoteItemView } from '@/classes/use-add-dnote-item-view'

const props = defineProps<AddDnoteItemViewProps>()
const emits = defineEmits<AddDnoteItemViewEmits>()

const {
    title,
    prefix,
    suffix,
    root_predicate,
    aggregate_targets,
    aggregate_target,
    reset,
    save,
} = useAddDnoteItemView({ props, emits })
</script>