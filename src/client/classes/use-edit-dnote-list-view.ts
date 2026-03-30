'use strict'

import { nextTick, ref, type Ref } from 'vue'
import type PredicateGroupType from '@/classes/dnote/predicate-group-type'
import type Predicate from '@/classes/dnote/predicate'
import aggregate_target_menu_items from '@/classes/dnote/pulldown-menu/aggregate-target-menu-items'
import kyou_getter_menu_items from '@/classes/dnote/pulldown-menu/kyou-getter-menu-items'
import type DnoteSelectItem from '@/classes/dnote/dnote-select-item'
import DnoteListQuery from '@/pages/views/dnote-list-query'
import type EditDnoteListViewEmits from '@/pages/views/edit-dnote-list-view-emits'
import type EditDnoteListViewProps from '@/pages/views/edit-dnote-list-view-props'
import { build_dnote_aggregate_target_from_json, build_dnote_key_getter_from_json, build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'

export function useEditDnoteListView(options: {
    props: EditDnoteListViewProps
    emits: EditDnoteListViewEmits
}) {
    const { props, emits } = options

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

    const key_getters: Ref<Array<DnoteSelectItem>> = ref(kyou_getter_menu_items)
    const key_getter: Ref<string> = ref(key_getters.value[0].value)

    nextTick(() => load_props())

    async function load_props(): Promise<void> {
        id.value = props.dnote_list_query.id
        title.value = props.dnote_list_query.title
        prefix.value = props.dnote_list_query.prefix
        suffix.value = props.dnote_list_query.suffix
        root_predicate.value = predicate_struct_from_json(props.dnote_list_query.predicate.predicate_struct_to_json()) as PredicateGroupType
        key_getter.value = key_getters.value.find((key_getter) => key_getter.value === props.dnote_list_query.key_getter.to_json().type)!.value
        aggregate_target.value = aggregate_targets.value.find((aggregate_target) => aggregate_target.value === props.dnote_list_query.aggregate_target.to_json().type)!.value
    }

    async function reset(): Promise<void> {
        return load_props()
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

        emits('requested_update_dnote_list_query', new_dnote_list_query)
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

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
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

    return {
        id,
        title,
        prefix,
        suffix,
        root_predicate,
        aggregate_targets,
        aggregate_target,
        key_getters,
        key_getter,
        reset,
        save,
    }
}
