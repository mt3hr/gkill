import type { ModelRef } from 'vue'
import type PredicateGroupType from '@/classes/dnote/predicate-group-type'
import type Predicate from '@/classes/dnote/predicate'
import predicate_menu_items from '@/classes/dnote/pulldown-menu/predicate-menu-items'

export function useEditDnotePredicateGroup(options: {
    group: ModelRef<PredicateGroupType | undefined>,
}) {
    const { group } = options

    // ── Methods ──
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

    // ── Return ──
    return {
        // Methods
        is_group,
        add_predicate,
        add_group,
        remove_predicate,
    }
}
