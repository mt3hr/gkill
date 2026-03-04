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
import { nextTick, ref, type Ref } from 'vue'
import PredicateGroup from './edit-dnote-predicate-group.vue'
import type PredicateGroupType from '../../classes/dnote/predicate-group-type'
import type Predicate from '../../classes/dnote/predicate'
import aggregate_target_menu_items from '@/classes/dnote/pulldown-menu/aggregate-target-menu-items'
import kyou_getter_menu_items from '@/classes/dnote/pulldown-menu/kyou-getter-menu-items'
import type DnoteSelectItem from '../../classes/dnote/dnote-select-item'
import DnoteListQuery from '@/pages/views/dnote-list-query'
import type AddDnoteListViewEmits from './add-dnote-list-view-emits'
import type AddDnoteListViewProps from './add-dnote-list-view-props'
import { build_dnote_aggregate_target_from_json, build_dnote_key_getter_from_json, build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'

const props = defineProps<AddDnoteListViewProps>()
const emits = defineEmits<AddDnoteListViewEmits>()

nextTick(() => reset())

async function reset(): Promise<void> {
    id.value = props.gkill_api.generate_uuid()
    title.value = ""
    prefix.value = ""
    suffix.value = ""
    root_predicate.value = {
        logic: 'AND',
        predicates: []
    }
    key_getter.value = key_getters.value[0].value
    aggregate_target.value = aggregate_targets.value[0].value

}

async function save(): Promise<void> {
    const new_dnote_list_query = new DnoteListQuery()
    new_dnote_list_query.id = id.value
    new_dnote_list_query.prefix = prefix.value
    new_dnote_list_query.suffix = suffix.value
    new_dnote_list_query.title = title.value
    new_dnote_list_query.aggregate_target = build_dnote_aggregate_target_from_json({ type: aggregate_target.value })
    new_dnote_list_query.key_getter = build_dnote_key_getter_from_json({ type: key_getter.value })
    new_dnote_list_query.predicate = build_dnote_predicate_from_json(predicate_struct_to_json(root_predicate.value))

    emits('requested_add_dnote_list_query', new_dnote_list_query)
    emits('requested_close_dialog')
}

const id = ref("")
const title = ref("")
const prefix = ref("")
const suffix = ref("")

const root_predicate = ref<PredicateGroupType>({
    logic: 'AND',
    predicates: []
})

const aggregate_targets: Ref<Array<DnoteSelectItem>> = ref(aggregate_target_menu_items)
const aggregate_target: Ref<string> = ref(aggregate_targets.value[0].value)

const key_getters: Ref<Array<DnoteSelectItem>> = ref(kyou_getter_menu_items)
const key_getter: Ref<string> = ref(key_getters.value[0].value)

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

function is_group(p: Predicate | PredicateGroupType): p is PredicateGroupType {
    return 'logic' in p && Array.isArray(p.predicates)
}
</script>
