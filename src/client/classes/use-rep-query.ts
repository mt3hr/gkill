import { type Ref, ref, watch, nextTick } from 'vue'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { CheckState } from '@/pages/views/check-state'
import { apply_check_state_to_struct } from '@/classes/foldable-struct-check'
import { deep_equals } from '@/classes/deep-equals'
import type { RepQueryEmits } from '@/pages/views/rep-query-emits'
import type { RepQueryProps } from '@/pages/views/rep-query-props'
import type FoldableStruct from '@/pages/views/foldable-struct.vue'

export function useRepQuery(options: {
    props: RepQueryProps,
    emits: RepQueryEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const foldable_struct_reps = ref<InstanceType<typeof FoldableStruct> | null>(null)
    const foldable_struct_devices = ref<InstanceType<typeof FoldableStruct> | null>(null)
    const foldable_struct_rep_types = ref<InstanceType<typeof FoldableStruct> | null>(null)

    // ── State refs ──
    const cloned_query: Ref<FindKyouQuery> = ref(props.find_kyou_query.clone())
    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

    const tab = ref(2)
    const use_rep = ref(true)

    const loading = ref(false)
    const skip_emits_this_tick = ref(false)

    // ── Watchers ──
    // loading の watcher は置かない。かつては loading=true でここでも3ツリーを
    // 更新していたが、find_kyou_query watcher が直後に行う処理と完全に同一で、
    // 数百repの環境では列フォーカス切替のフリーズを倍加させるだけだった。
    // loading 自体は update_check_devices/_rep_types 内の calc_reps 再入抑止に使う。

    watch(() => props.application_config, async (_new_application_config: ApplicationConfig, _old_application_config: ApplicationConfig) => {
        cloned_application_config.value = props.application_config.clone()
        if (props.inited) {
            skip_emits_this_tick.value = true
            nextTick(() => skip_emits_this_tick.value = false)
            update_check_devices(cloned_query.value.devices_in_sidebar, CheckState.checked, true, true)
            update_check_rep_types(cloned_query.value.rep_types_in_sidebar, CheckState.checked, true, true)
            // reps は null（フィルタ未使用）でありうる。deep_equals 側と同じく [] へ吸収する
            update_check_reps(cloned_query.value.reps ?? [], CheckState.checked, true, true)
            return
        }
        if (!props.inited) {
            emits('inited')
        }
    })

    watch(() => props.find_kyou_query, async (new_value: FindKyouQuery, _old_value: FindKyouQuery) => {
        if (!new_value) {
            return
        }
        loading.value = true
        try {
            cloned_query.value = new_value.clone()
            const reps = cloned_query.value.reps
            const devices = cloned_query.value.devices_in_sidebar
            const rep_types = cloned_query.value.rep_types_in_sidebar
            if (devices) {
                await update_check_devices(devices, CheckState.checked, true, true)
            }
            if (rep_types) {
                await update_check_rep_types(rep_types, CheckState.checked, true, true)
            }
            if (reps) {
                await update_check_reps(reps, CheckState.checked, true, true)
            }

            const checked_reps = foldable_struct_reps.value?.get_selected_items()
            if (checked_reps) {
                emits('request_update_checked_reps', checked_reps, false)
            }
            const checked_devices = foldable_struct_devices.value?.get_selected_items()
            if (checked_devices) {
                emits('request_update_checked_devices', checked_devices, false)
            }
            const checked_rep_types = foldable_struct_rep_types.value?.get_selected_items()
            if (checked_rep_types) {
                emits('request_update_checked_rep_types', checked_rep_types, false)
            }
        } finally {
            // 例外時も必ずloadingを戻す。戻さないと update_check_devices/_rep_types の
            // 再計算ガード(if (!loading.value))が二度と通らず、
            // プロファイル/記録分類→記録先詳細の算出だけが無言で死ぬ
            loading.value = false
        }
    })

    // ── Business logic ──
    function calc_reps_by_types_and_devices(): Array<string> | null {
        const query_for_apply_summary_to_detail = cloned_query.value.clone()
        query_for_apply_summary_to_detail.apply_rep_summary_to_detaul(cloned_application_config.value)
        return query_for_apply_summary_to_detail.reps
    }

    function clicked_reps(e: MouseEvent, items: Array<string>, is_checked: CheckState) {
        update_check_reps(items, is_checked, true)
    }

    function update_reps(items: Array<string>, is_checked: CheckState) {
        update_check_reps(items, is_checked, false)
    }

    function clicked_devices(e: MouseEvent, items: Array<string>, is_checked: CheckState) {
        update_check_devices(items, is_checked, true)
    }

    function update_devices(items: Array<string>, is_checked: CheckState) {
        update_check_devices(items, is_checked, false)
    }

    function clicked_rep_types(e: MouseEvent, items: Array<string>, is_checked: CheckState) {
        update_check_rep_types(items, is_checked, true)
    }

    function update_rep_types(items: Array<string>, is_checked: CheckState) {
        update_check_rep_types(items, is_checked, false)
    }

    function update_check_reps(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean, disable_emits?: boolean) {
        apply_check_state_to_struct(cloned_application_config.value.rep_struct, items, is_checked, pre_uncheck_all)
        const reps = foldable_struct_reps.value?.get_selected_items()
        // reps は null（フィルタ未使用）でありうる。null と [] の差で
        // 機械的同期のemit抑止が壊れないよう [] へ吸収して比較する
        if (reps && !deep_equals(reps, cloned_query.value?.reps ?? [])) {
            if (!skip_emits_this_tick.value && !disable_emits) {
                emits('request_update_checked_reps', reps, true)
            }
        }
        foldable_struct_reps.value?.update_check()
    }

    function update_check_devices(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean, disable_emits?: boolean) {
        apply_check_state_to_struct(cloned_application_config.value.device_struct, items, is_checked, pre_uncheck_all)

        const devices = foldable_struct_devices.value?.get_selected_items()
        if (devices && !deep_equals(devices, cloned_query.value?.devices_in_sidebar ?? [])) {
            if (!skip_emits_this_tick.value && !disable_emits) {
                emits('request_update_checked_devices', devices, true)
            }
        }
        if (!loading.value) {
            const reps = calc_reps_by_types_and_devices()
            if (reps && !deep_equals(reps, cloned_query.value?.reps ?? [])) {
                update_check_reps(reps, CheckState.checked, true, disable_emits)
            }
        }
        foldable_struct_devices.value?.update_check()
    }

    function update_check_rep_types(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean, disable_emits?: boolean) {
        apply_check_state_to_struct(cloned_application_config.value.rep_type_struct, items, is_checked, pre_uncheck_all)

        const rep_types = foldable_struct_rep_types.value?.get_selected_items()
        if (rep_types && !deep_equals(rep_types, cloned_query.value?.rep_types_in_sidebar ?? [])) {
            if (!skip_emits_this_tick.value && !disable_emits) {
                emits('request_update_checked_rep_types', rep_types, true)
            }
        }
        if (!loading.value) {
            const reps = calc_reps_by_types_and_devices()
            if (reps && !deep_equals(reps, cloned_query.value?.reps ?? [])) {
                update_check_reps(reps, CheckState.checked, true, disable_emits)
            }
        }
        foldable_struct_rep_types.value?.update_check()
    }

    function get_checked_reps(): Array<string> | null {
        const reps = foldable_struct_reps.value?.get_selected_items()
        if (!reps) {
            return null
        }
        return reps
    }

    function get_checked_devices(): Array<string> | null {
        const devices = foldable_struct_devices.value?.get_selected_items()
        if (!devices) {
            return null
        }
        return devices
    }

    function get_checked_rep_types(): Array<string> | null {
        const rep_types = foldable_struct_rep_types.value?.get_selected_items()
        if (!rep_types) {
            return null
        }
        return rep_types
    }

    // ── Return ──
    return {
        // Template refs
        foldable_struct_reps,
        foldable_struct_devices,
        foldable_struct_rep_types,

        // State
        cloned_application_config,
        tab,
        use_rep,

        // Business logic / exposed
        get_checked_reps,
        get_checked_devices,
        get_checked_rep_types,
        update_check_devices,
        update_check_rep_types,
        update_check_reps,

        // Template event handlers
        clicked_reps,
        update_reps,
        clicked_devices,
        update_devices,
        clicked_rep_types,
        update_rep_types,
    }
}
