<!-- PredicateCard.vue -->
<template>
    <v-card class="predicate-card">
        <v-select v-model="model_value!.type" :items="predicate_types" :label="i18n.global.t('DNOTE_PREDICATE_TITLE')"
            item-title="label" item-value="value" />
        <v-select v-if="model_value!.type === 'DataTypePrefixPredicate' || model_value!.type === 'EqualDataTypeTargetKyouPredicate'" v-model="model_value!.value"
            :items="rep_type_items" item-title="label" item-value="value"
            :label="i18n.global.t('DNOTE_PREDICATE_VALUE_TITLE')" />
        <v-text-field v-else v-model="model_value!.value"
            :label="i18n.global.t('DNOTE_PREDICATE_VALUE_TITLE')" />
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn :color="'secondary'" @click="emits('remove')" class="mt-2">{{ i18n.global.t('DELETE_TITLE') }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { useEditDnoteCard } from '@/classes/use-edit-dnote-card'
import type Predicate from '../../classes/dnote/predicate'

const emits = defineEmits(['remove'])
const model_value = defineModel<Predicate>()

const {
    predicate_types,
    rep_type_items,
} = useEditDnoteCard()
</script>
