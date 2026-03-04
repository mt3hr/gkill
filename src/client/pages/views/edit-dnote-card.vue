<!-- PredicateCard.vue -->
<template>
    <v-card class="predicate-card">
        <v-select v-model="model_value!.type" :items="predicate_types" :label="i18n.global.t('DNOTE_PREDICATE_TITLE')"
            item-title="label" item-value="value" />
        <v-text-field v-model="model_value!.value" :label="i18n.global.t('DNOTE_PREDICATE_VALUE_TITLE')" />
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
const emits = defineEmits(['remove'])
import { ref, type Ref } from 'vue'
import { defineModel } from 'vue'
import type Predicate from '../../classes/dnote/predicate'
import predicate_menu_items from '@/classes/dnote/pulldown-menu/predicate-menu-items'
import type DnoteSelectItem from '../../classes/dnote/dnote-select-item'

const model_value = defineModel<Predicate>()

const predicate_types: Ref<Array<DnoteSelectItem>> = ref(predicate_menu_items)
</script>
