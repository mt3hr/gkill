<template>
    <v-card class="pa-2">
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
import { nextTick, ref, type Ref } from 'vue'
import PredicateGroup from './edit-dnote-predicate-group.vue'
import type PredicateGroupType from '../../classes/dnote/predicate-group-type'
import type Predicate from '../../classes/dnote/predicate'
import type AddRyuuItemViewEmits from './add-ryuu-item-view-emits'
import type AddRyuuItemViewProps from './add-ryuu-item-view-props'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query'
import { RelatedTimeMatchType } from '@/classes/dnote/related-time-match-type'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'
import FindQueryEditorDialog from '../dialogs/find-query-editor-dialog.vue'

const find_query_editor_dialog = ref<InstanceType<typeof FindQueryEditorDialog> | null>(null);

const props = defineProps<AddRyuuItemViewProps>()
const emits = defineEmits<AddRyuuItemViewEmits>()

const id = ref("")
const title = ref("")
const prefix = ref("")
const suffix = ref("")
const related_time_match_type = ref(RelatedTimeMatchType.NEAR_RELATED_TIME)
const find_kyou_query: Ref<FindKyouQuery | null> = ref(null)
const find_duration_hour = ref(1)

const is_use_custom_find_kyou_query = ref(false)

const related_time_match_types = ref([
    { label: i18n.global.t('RYUU_RELATED_NEAR'), value: RelatedTimeMatchType.NEAR_RELATED_TIME },
    { label: i18n.global.t('RYUU_RELATED_NEAR_BEFORE'), value: RelatedTimeMatchType.NEAR_RELATED_TIME_BEFORE },
    { label: i18n.global.t('RYUU_RELATED_NEAR_AFTER'), value: RelatedTimeMatchType.NEAR_RELATED_TIME_AFTER },
])

const root_predicate = ref<PredicateGroupType>({
    logic: 'AND',
    predicates: []
})

async function reset(): Promise<void> {
    id.value = props.gkill_api.generate_uuid()
    title.value = ""
    prefix.value = ""
    suffix.value = ""
    root_predicate.value = {
        logic: 'AND',
        predicates: []
    }
    related_time_match_type.value = RelatedTimeMatchType.NEAR_RELATED_TIME
    is_use_custom_find_kyou_query.value = false
    find_kyou_query.value = null
    find_duration_hour.value = 1
}

async function save(): Promise<void> {
    const new_related_kyou_query = new RelatedKyouQuery()
    new_related_kyou_query.id = id.value
    new_related_kyou_query.prefix = prefix.value
    new_related_kyou_query.suffix = suffix.value
    new_related_kyou_query.title = title.value
    new_related_kyou_query.predicate = build_dnote_predicate_from_json(predicate_struct_to_json(root_predicate.value))
    new_related_kyou_query.related_time_match_type = related_time_match_type.value
    new_related_kyou_query.find_kyou_query = is_use_custom_find_kyou_query.value ? find_kyou_query.value : null
    new_related_kyou_query.find_duration_hour = find_duration_hour.value

    emits('requested_add_related_kyou_query', new_related_kyou_query)
    emits('requested_close_dialog')
}

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

function show_find_query_editor_dialog(): void {
    if (!find_kyou_query.value) {
        find_kyou_query.value = FindKyouQuery.generate_default_query_for_rykv(props.application_config)
    }
    const cloned_find_kyou_query = find_kyou_query.value.clone()
    nextTick(() => {
        find_query_editor_dialog.value?.show(cloned_find_kyou_query)
    })
}

nextTick(() => reset())
</script>
<style lang="css" scoped></style>