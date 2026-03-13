<template>
    <v-card class="predicate-group" variant="outlined">
        <v-select v-model="group!.logic" :items="['AND', 'OR', 'NOT']" :label="i18n.global.t('DNOTE_JOIN_LOGIC_TITLE')" />

        <div v-for="(predicate, index) in group!.predicates" :key="index" class="predicate-entry">
            <PredicateGroup v-if="is_group(predicate)" v-model="group!.predicates[index] as PredicateGroupType"
                @remove="remove_predicate(index)" />
            <PredicateCard v-else v-model="group!.predicates[index] as Predicate" @remove="remove_predicate(index)" />
        </div>
        <v-btn @click="add_predicate" :color="'primary'" class="ma-1">{{ i18n.global.t('DNOTE_ADD_PREDICATE_TITLE') }}</v-btn>
        <v-btn @click="add_group" :color="'primary'" class="ma-1">{{ i18n.global.t('DNOTE_ADD_GROUP_TITLE') }}</v-btn>
        <v-btn @click="emits('remove')" :color="'secondary'" class="ma-1"> {{ i18n.global.t('DNOTE_DELETE_GROUP_TITLE') }}</v-btn>
    </v-card>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
const emits = defineEmits(['remove'])
import { defineModel } from 'vue'
import PredicateCard from './edit-dnote-card.vue'
import PredicateGroup from './edit-dnote-predicate-group.vue'
import type Predicate from '../../classes/dnote/predicate'
import type PredicateGroupType from '../../classes/dnote/predicate-group-type'
import { useEditDnotePredicateGroup } from '@/classes/use-edit-dnote-predicate-group'

const group = defineModel<PredicateGroupType>()!

const {
    is_group,
    add_predicate,
    add_group,
    remove_predicate,
} = useEditDnotePredicateGroup({ group })
</script>
