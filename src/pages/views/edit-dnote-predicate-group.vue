<template>
    <v-card class="predicate-group" variant="outlined">
        <v-select v-model="group!.logic" :items="['AND', 'OR', 'NOT']" :label="$t('DNOTE_JOIN_LOGIC_TITLE')" />

        <div v-for="(predicate, index) in group!.predicates" :key="index" class="predicate-entry">
            <PredicateGroup v-if="is_group(predicate)" v-model="group!.predicates[index] as PredicateGroupType"
                @remove="remove_predicate(index)" />
            <PredicateCard v-else v-model="group!.predicates[index] as Predicate" @remove="remove_predicate(index)" />
        </div>
        <v-btn @click="add_predicate" :color="'primary'" class="ma-1">{{ $t('DNOTE_ADD_PREDICATE_TITLE') }}</v-btn>
        <v-btn @click="add_group" :color="'primary'" class="ma-1">{{ $t('DNOTE_ADD_GROUP_TITLE') }}</v-btn>
        <v-btn @click="emits('remove')" :color="'secondary'" class="ma-1"> {{ $t('DNOTE_DELETE_GROUP_TITLE') }}</v-btn>
    </v-card>
</template>

<script setup lang="ts">
const emits = defineEmits(['remove'])
import { defineModel } from 'vue'
import PredicateCard from './edit-dnote-card.vue'
import PredicateGroup from './edit-dnote-predicate-group.vue'
import type Predicate from '../../classes/dnote/predicate'
import type PredicateGroupType from '../../classes/dnote/predicate-group-type'
import predicate_menu_items from '@/classes/dnote/pulldown-menu/predicate-menu-items'

const group = defineModel<PredicateGroupType>()!

function is_group(p: Predicate | PredicateGroupType): p is PredicateGroupType {
    return 'logic' in p && Array.isArray(p.predicates)
}

function add_predicate() {
    group.value!.predicates.push({ type: predicate_menu_items[0].value, value: "" })
}

function add_group() {
    group.value!.predicates.push({ logic: 'AND', predicates: [] })
}

function remove_predicate(index: number) {
    group.value!.predicates.splice(index, 1)
}
</script>