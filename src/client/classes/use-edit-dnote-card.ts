import { ref, type Ref } from 'vue'
import predicate_menu_items from '@/classes/dnote/pulldown-menu/predicate-menu-items'
import rep_type_menu_items from '@/classes/dnote/pulldown-menu/rep-type-menu-items'
import type DnoteSelectItem from '@/classes/dnote/dnote-select-item'

export function useEditDnoteCard() {
    const predicate_types: Ref<Array<DnoteSelectItem>> = ref(predicate_menu_items)
    const rep_type_items: Ref<Array<DnoteSelectItem>> = ref(rep_type_menu_items)

    return {
        predicate_types,
        rep_type_items,
    }
}
