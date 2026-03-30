import { nextTick, ref, type Ref } from 'vue'
import type PredicateGroupType from '@/classes/dnote/predicate-group-type'
import type Predicate from '@/classes/dnote/predicate'
import aggregate_target_menu_items from '@/classes/dnote/pulldown-menu/aggregate-target-menu-items'
import kyou_getter_menu_items from '@/classes/dnote/pulldown-menu/kyou-getter-menu-items'
import type DnoteSelectItem from '@/classes/dnote/dnote-select-item'
import DnoteListQuery from '@/pages/views/dnote-list-query'
import type AddDnoteListViewEmits from '@/pages/views/add-dnote-list-view-emits'
import type AddDnoteListViewProps from '@/pages/views/add-dnote-list-view-props'
import { build_dnote_aggregate_target_from_json, build_dnote_key_getter_from_json, build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'

export function useAddDnoteListView(options: {
    props: AddDnoteListViewProps,
    emits: AddDnoteListViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
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

    // ── Init ──
    nextTick(() => reset())

    // ── Business logic ──
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

    function predicate_struct_to_json(group: PredicateGroupType | Predicate): Record<string, unknown> {
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

    // ── Return ──
    return {
        // State
        id,
        title,
        prefix,
        suffix,
        root_predicate,
        aggregate_targets,
        aggregate_target,
        key_getters,
        key_getter,

        // Business logic
        reset,
        save,
    }
}
