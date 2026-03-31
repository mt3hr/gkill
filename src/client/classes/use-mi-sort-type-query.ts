'use strict'

import { nextTick, type Ref, ref, watch } from 'vue'
import { i18n } from '@/i18n'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import type { miSortTypeQueryProps } from '@/pages/views/mi-sort-type-query-props'
import type { miSortTypeQueryEmits } from '@/pages/views/mi-sort-type-query-emits'

export function useMiSortTypeQuery(options: { props: miSortTypeQueryProps, emits: miSortTypeQueryEmits }) {
    const { props, emits } = options

    const query = ref(props.find_kyou_query.clone())
    const skip_emits_for_prop_change = ref(false)

    watch(() => props.find_kyou_query, () => {
        if (!props.find_kyou_query) {
            return
        }
        query.value = props.find_kyou_query.clone()
        skip_emits_for_prop_change.value = true
        load_sort_type()
        nextTick(() => skip_emits_for_prop_change.value = false)
    })

    nextTick(() => {
        load_sort_type()
        emits('inited')
    })

    const use_sort_type = ref(true)
    const sort_type: Ref<MiSortType> = ref(MiSortType.create_time)

    watch(() => sort_type.value, () => {
        if (!skip_emits_for_prop_change.value) {
            emits('request_update_sort_type', sort_type.value)
        }
    })

    function load_sort_type(): void {
        for (let i = 0; i < sort_types.value.length; i++) {
            if (sort_types.value[i].value === query.value.mi_sort_type) {
                sort_type.value = sort_types.value[i].value
                break
            }
        }
    }

    const sort_types: Ref<Array<{ name: string, value: MiSortType }>> = ref([
        {
            name: i18n.global.t("MI_CREATE_DATE_TIME_TITLE"),
            value: MiSortType.create_time,
        },
        {
            name: i18n.global.t("MI_START_DATE_TIME_TITLE"),
            value: MiSortType.estimate_start_time,
        },
        {
            name: i18n.global.t("MI_END_DATE_TIME_TITLE"),
            value: MiSortType.estimate_end_time,
        },
        {
            name: i18n.global.t("MI_LIMIT_DATE_TIME_TITLE"),
            value: MiSortType.limit_time,
        }
    ])

    function get_sort_type(): MiSortType {
        return sort_type.value
    }

    return {
        query,
        use_sort_type,
        sort_type,
        sort_types,
        get_sort_type,
    }
}
