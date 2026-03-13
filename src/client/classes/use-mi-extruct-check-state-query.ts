import { i18n } from '@/i18n'
import { nextTick, type Ref, ref, watch } from 'vue'
import { MiCheckState } from '@/classes/api/find_query/mi-check-state'
import type { miExtructCheckStateQueryEmits } from '@/pages/views/mi-extruct-check-state-query-emits'
import type { miExtructCheckStateQueryProps } from '@/pages/views/mi-extruct-check-state-query-props'

export function useMiExtructCheckStateQuery(options: {
    props: miExtructCheckStateQueryProps,
    emits: miExtructCheckStateQueryEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const query = ref(props.find_kyou_query.clone())
    const use_mi_check_state = ref(true)
    const check_state: Ref<MiCheckState> = ref(MiCheckState.uncheck)

    const check_states: Ref<Array<{ name: string, value: MiCheckState }>> = ref([
        {
            name: i18n.global.t("MI_CHECK_STATE_ALL_TITLE"),
            value: MiCheckState.all,
        },
        {
            name: i18n.global.t("MI_CHECK_STATE_CHECKED_TITLE"),
            value: MiCheckState.checked,
        },
        {
            name: i18n.global.t("MI_CHECK_STATE_UNCHECKED_TITLE"),
            value: MiCheckState.uncheck,
        }
    ])

    // ── Watchers ──
    watch(() => props.find_kyou_query, () => {
        if (!props.find_kyou_query) {
            return
        }
        query.value = props.find_kyou_query.clone()
        load_check_state()
    })

    watch(() => check_state.value, () => {
        emits('request_update_extruct_check_state', check_state.value)
    })

    // ── Lifecycle ──
    nextTick(() => {
        load_check_state()
        emits('inited')
    })

    // ── Methods ──
    function load_check_state(): void {
        for (let i = 0; i < check_states.value.length; i++) {
            if (query.value.mi_check_state === check_states.value[i].value) {
                check_state.value = check_states.value[i].value
                break
            }
        }
    }

    function get_update_extruct_check_state(): MiCheckState {
        return check_state.value
    }

    // ── Return ──
    return {
        // State
        query,
        use_mi_check_state,
        check_state,
        check_states,

        // Methods
        load_check_state,
        get_update_extruct_check_state,
    }
}
