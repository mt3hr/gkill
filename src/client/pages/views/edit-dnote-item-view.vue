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
import { nextTick, ref, type Ref } from 'vue'
import PredicateGroup from './edit-dnote-predicate-group.vue'
import type PredicateGroupType from '../../classes/dnote/predicate-group-type'
import type Predicate from '../../classes/dnote/predicate'
import aggregate_target_menu_items from '@/classes/dnote/pulldown-menu/aggregate-target-menu-items'
import type DnoteSelectItem from '../../classes/dnote/dnote-select-item'
import type EditDnoteItemViewEmits from './edit-dnote-item-view-emits'
import type EditDnoteItemViewProps from './edit-dnote-item-view-props'
import DnoteItem from '@/classes/dnote/dnote-item'
import { build_dnote_aggregate_target_from_json, build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'

const props = defineProps<EditDnoteItemViewProps>()
const emits = defineEmits<EditDnoteItemViewEmits>()
const model_value = defineModel<DnoteItem>()

nextTick(() => { load_props() })

async function load_props(): Promise<void> {
    id.value = model_value.value!.id
    title.value = model_value.value!.title
    prefix.value = model_value.value!.prefix
    suffix.value = model_value.value!.suffix
    root_predicate.value = predicate_struct_from_json(model_value.value!.predicate.predicate_struct_to_json()) as PredicateGroupType
    aggregate_target.value = aggregate_targets.value.find((aggregate_target) => aggregate_target.value === model_value.value!.agregate_target.to_json().type)!.value
}

async function reset(): Promise<void> {
    return load_props()
}

async function save(): Promise<void> {
    const new_dnote_item = new DnoteItem()
    new_dnote_item.id = id.value
    new_dnote_item.prefix = prefix.value
    new_dnote_item.suffix = suffix.value
    new_dnote_item.title = title.value
    new_dnote_item.agregate_target = build_dnote_aggregate_target_from_json({ type: aggregate_target.value })
    new_dnote_item.predicate = build_dnote_predicate_from_json(predicate_struct_to_json(root_predicate.value))

    emits('requested_update_dnote_item', new_dnote_item)
    emits('requested_close_dialog')
}

const id = ref(props.gkill_api.generate_uuid())
const title = ref("")
const prefix = ref("")
const suffix = ref("")

const root_predicate = ref<PredicateGroupType>({
    logic: 'AND',
    predicates: []
})

const aggregate_targets: Ref<Array<DnoteSelectItem>> = ref(aggregate_target_menu_items)
const aggregate_target: Ref<string> = ref(aggregate_targets.value[0].value)

function predicate_struct_to_json(group: PredicateGroupType | Predicate): any {
    if (is_group(group)) {
        return {
            logic: group.logic,
            predicates: group.predicates.map(p => predicate_struct_to_json(p))
        }
    } else {
        return { type: group.type, value: group.value }
    }
}

function predicate_struct_from_json(json: any): PredicateGroupType | Predicate {
    if (json.logic && Array.isArray(json.predicates)) {
        return {
            logic: json.logic,
            predicates: json.predicates.map(predicate_struct_from_json)
        }
    } else {
        return {
            type: json.type,
            value: json.value
        }
    }
}

function is_group(p: Predicate | PredicateGroupType): p is PredicateGroupType {
    return 'logic' in p && Array.isArray(p.predicates)
}
</script>
