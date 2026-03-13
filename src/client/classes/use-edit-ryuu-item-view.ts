import { nextTick, ref, type Ref, type ModelRef } from 'vue'
import { i18n } from '@/i18n'
import type PredicateGroupType from '@/classes/dnote/predicate-group-type'
import type Predicate from '@/classes/dnote/predicate'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query'
import { RelatedTimeMatchType } from '@/classes/dnote/related-time-match-type'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'
import type FindQueryEditorDialog from '@/pages/dialogs/find-query-editor-dialog.vue'
import type EditRyuuItemViewEmits from '@/pages/views/edit-ryuu-item-view-emits'
import type EditRyuuItemViewProps from '@/pages/views/edit-ryuu-item-view-props'

export function useEditRyuuItemView(options: {
    props: EditRyuuItemViewProps,
    emits: EditRyuuItemViewEmits,
    model_value: ModelRef<RelatedKyouQuery | undefined>,
}) {
    const props = options.props
    const emits = options.emits
    const model_value = options.model_value

    // ── Template refs ──
    const find_query_editor_dialog = ref<InstanceType<typeof FindQueryEditorDialog> | null>(null)

    // ── State refs ──
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

    // ── Methods ──
    async function reset(): Promise<void> {
        id.value = model_value.value!.id
        title.value = model_value.value!.title
        prefix.value = model_value.value!.prefix
        suffix.value = model_value.value!.suffix
        root_predicate.value = model_value.value!.predicate.predicate_struct_to_json()
        related_time_match_type.value = model_value.value!.related_time_match_type
        is_use_custom_find_kyou_query.value = model_value.value!.find_kyou_query !== null
        find_kyou_query.value = model_value.value!.find_kyou_query
        find_duration_hour.value = model_value.value!.find_duration_hour
    }

    async function save(): Promise<void> {
        const updated_related_kyou_query = new RelatedKyouQuery()
        updated_related_kyou_query.id = model_value.value!.id
        updated_related_kyou_query.prefix = prefix.value
        updated_related_kyou_query.suffix = suffix.value
        updated_related_kyou_query.title = title.value
        updated_related_kyou_query.predicate = build_dnote_predicate_from_json(predicate_struct_to_json(root_predicate.value))
        updated_related_kyou_query.related_time_match_type = related_time_match_type.value
        updated_related_kyou_query.find_kyou_query = is_use_custom_find_kyou_query.value ? find_kyou_query.value : null
        updated_related_kyou_query.find_duration_hour = find_duration_hour.value

        model_value.value = updated_related_kyou_query
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
            find_kyou_query.value = new FindKyouQuery()
        }
        const cloned_find_kyou_query = find_kyou_query.value.clone()
        nextTick(() => {
            find_query_editor_dialog.value?.show(cloned_find_kyou_query)
        })
    }

    // ── Init ──
    nextTick(() => reset())

    return {
        // Template refs
        find_query_editor_dialog,

        // State
        id,
        title,
        prefix,
        suffix,
        related_time_match_type,
        find_kyou_query,
        find_duration_hour,
        is_use_custom_find_kyou_query,
        related_time_match_types,
        root_predicate,

        // Methods
        reset,
        save,
        show_find_query_editor_dialog,
    }
}
