import { nextTick, ref, type Ref } from 'vue'
import { i18n } from '@/i18n'
import type PredicateGroupType from '@/classes/dnote/predicate-group-type'
import type Predicate from '@/classes/dnote/predicate'
import type AddRyuuItemViewEmits from '@/pages/views/add-ryuu-item-view-emits'
import type AddRyuuItemViewProps from '@/pages/views/add-ryuu-item-view-props'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query'
import { RelatedTimeMatchType } from '@/classes/dnote/related-time-match-type'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'

export function useAddRyuuItemView(options: {
    props: AddRyuuItemViewProps,
    emits: AddRyuuItemViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const find_query_editor_dialog = ref()

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

    // ── Initialization ──
    nextTick(() => reset())

    // ── Return ──
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

        // Methods used in template
        reset,
        save,
        show_find_query_editor_dialog,
    }
}
