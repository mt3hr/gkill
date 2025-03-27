<template>
    <div class="replist">
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox readonly v-model="use_rep" label="記録保管場所" hide-details />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0">
                <v-btn dark color="secondary" @click="emits('request_clear_rep_query')" hide-details>クリア</v-btn>
            </v-col>
        </v-row>
        <v-tabs v-show="use_rep" v-model="tab">
            <v-tab key="summary">Summary</v-tab>
            <v-tab key="detail">Detail</v-tab>
        </v-tabs>
        <v-window v-model="tab">
            <v-window-item key="summary" :eager="true">
                <h2>Devices</h2>
                <table class="devicelist">
                    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                        :is_editable="false" :is_root="true" :is_show_checkbox="true" :is_open="false"
                        :struct_obj="cloned_application_config.parsed_device_struct"
                        @requested_update_check_state="update_devices"
                        @received_errors="(errors) => emits('received_errors', errors)"
                        @received_messages="(messages) => emits('received_messages', messages)"
                        @clicked_items="clicked_devices" ref="foldable_struct_devices" />
                </table>
                <h2>Types</h2>
                <table class="typelist">
                    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                        :is_editable="false" :is_root="true" :is_show_checkbox="true" :is_open="true"
                        :struct_obj="cloned_application_config.parsed_rep_type_struct"
                        @requested_update_check_state="update_rep_types"
                        @received_errors="(errors) => emits('received_errors', errors)"
                        @received_messages="(messages) => emits('received_messages', messages)"
                        @clicked_items="clicked_rep_types" ref="foldable_struct_rep_types" />
                </table>
            </v-window-item>
            <v-window-item key="detail" :eager="true">
                <h2>Reps</h2>
                <table>
                    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                        :is_editable="false" :is_root="true" :is_show_checkbox="true" :is_open="true"
                        :struct_obj="cloned_application_config.parsed_rep_struct"
                        @requested_update_check_state="update_reps"
                        @received_errors="(errors) => emits('received_errors', errors)"
                        @received_messages="(messages) => emits('received_messages', messages)"
                        @clicked_items="clicked_reps" ref="foldable_struct_reps" />
                </table>
            </v-window-item>
        </v-window>
    </div>
</template>
<script setup lang="ts">
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { nextTick, type Ref, ref, watch } from 'vue'
import FoldableStruct from './foldable-struct.vue'
import type { RepQueryEmits } from './rep-query-emits'
import type { RepQueryProps } from './rep-query-props'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import { CheckState } from './check-state'
import type { FoldableStructModel } from './foldable-struct-model'
import { deepEquals } from '@/classes/deep-equals'

const foldable_struct_reps = ref<InstanceType<typeof FoldableStruct> | null>(null)
const foldable_struct_devices = ref<InstanceType<typeof FoldableStruct> | null>(null)
const foldable_struct_rep_types = ref<InstanceType<typeof FoldableStruct> | null>(null)

const props = defineProps<RepQueryProps>()
const emits = defineEmits<RepQueryEmits>()
defineExpose({ get_checked_reps, get_checked_devices, get_checked_rep_types })

const old_cloned_query: Ref<FindKyouQuery | null> = ref(null)
const cloned_query: Ref<FindKyouQuery> = ref(props.find_kyou_query.clone())
const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())


const tab = ref(2)
const use_rep = ref(true)

const loading = ref(false)
watch(() => loading.value, async (new_value: boolean, old_value: boolean) => {
    if (new_value !== old_value && new_value) {
        const reps = cloned_query.value.reps
        const devices = cloned_query.value.devices_in_sidebar
        const rep_types = cloned_query.value.rep_types_in_sidebar
        if (devices) {
            await update_check_devices(devices, CheckState.checked, true)
        }
        if (rep_types) {
            await update_check_rep_types(rep_types, CheckState.checked, true)
        }
        if (reps) {
            await update_check_reps(reps, CheckState.checked, true)
        }
    }
})

const skip_emits_this_tick = ref(false)
watch(() => props.application_config, async (_new_application_config: ApplicationConfig, old_application_config: ApplicationConfig) => {
    cloned_application_config.value = props.application_config.clone()
    const errors = await cloned_application_config.value.load_all()
    if (errors !== null && errors.length !== 0) {
        emits('received_errors', errors)
    }
    if (props.inited) {
        skip_emits_this_tick.value = true
        nextTick(() => skip_emits_this_tick.value = false)
        update_check_devices(cloned_query.value.devices_in_sidebar, CheckState.checked, true)
        update_check_rep_types(cloned_query.value.rep_types_in_sidebar, CheckState.checked, true)
        update_check_reps(cloned_query.value.reps, CheckState.checked, true)
        return
    }
    const reps = new Array<string>()
    const devices = new Array<string>()
    const rep_types = new Array<string>()
    cloned_application_config.value.rep_struct.forEach(rep => {
        if (rep.check_when_inited) {
            reps.push(rep.rep_name)
        }
    })
    cloned_application_config.value.device_struct.forEach(device => {
        if (device.check_when_inited) {
            devices.push(device.device_name)
        }
    })
    cloned_application_config.value.rep_type_struct.forEach(rep_type => {
        if (rep_type.check_when_inited) {
            rep_types.push(rep_type.rep_type_name)
        }
    })
    await update_check_devices(devices, CheckState.checked, true)
    await update_check_reps(rep_types, CheckState.checked, true)

    const calclated_reps = calc_reps_by_types_and_devices()
    if (!calclated_reps) {
        await update_check_reps(reps, CheckState.checked, true)
    } else {
        await update_check_reps(calclated_reps, CheckState.checked, true)
    }
    emits('request_update_checked_reps', reps, true)
    if (!props.inited) {
        emits('inited')
    }
})

watch(() => props.find_kyou_query, async (new_value: FindKyouQuery, old_value: FindKyouQuery) => {
    loading.value = true
    cloned_query.value = new_value.clone()
    old_cloned_query.value = old_value
    const reps = cloned_query.value.reps
    const devices = cloned_query.value.devices_in_sidebar
    const rep_types = cloned_query.value.rep_types_in_sidebar
    if (devices) {
        await update_check_devices(devices, CheckState.checked, true)
    }
    if (rep_types) {
        await update_check_rep_types(rep_types, CheckState.checked, true)
    }
    if (reps) {
        await update_check_reps(reps, CheckState.checked, true)
    }
    loading.value = false
})

// 現在チェックされているdevices, typesに該当するrep.nameを抽出してthis.repsを更新し、emitします。
function calc_reps_by_types_and_devices(): Array<string> | null {
    const reps = cloned_application_config.value.parsed_rep_struct.children
    const rep_types = cloned_application_config.value.parsed_rep_type_struct.children
    const devices = cloned_application_config.value.parsed_device_struct.children

    if (!reps || !devices || !rep_types) {
        return null
    }

    const check_target_rep_names = new Array<string>()
    let walk_rep = (_rep: RepStructElementData): void => { }
    walk_rep = (rep: RepStructElementData): void => {
        rep.is_checked = false
        const rep_struct = rep_to_struct(rep)

        let type_is_match = false
        let device_is_match = false
        let walk = (_struct: FoldableStructModel): void => { }

        walk = (struct: FoldableStructModel): void => {
            struct.indeterminate = false
            if (struct.is_checked && struct.key == rep_struct.type) {
                if (!type_is_match) {
                    type_is_match = true
                }
            }
            if (struct.children) {
                struct.children.forEach(child => walk(child))
            }
        }
        rep_types.forEach(rep_type => walk(rep_type))

        walk = (struct: FoldableStructModel): void => {
            struct.indeterminate = false
            if (struct.is_checked && struct.key == rep_struct.device) {
                if (!device_is_match) {
                    device_is_match = true
                }
            }
            if (struct.children) {
                struct.children.forEach(child => walk(child))
            }
        }
        devices.forEach(device => walk(device))

        if (type_is_match && device_is_match && !rep.ignore_check_rep_rykv) {
            check_target_rep_names.push(rep.rep_name)
        }

        if (rep.children) {
            rep.children.forEach(child_rep => walk_rep(child_rep))
        }
    }
    reps.forEach(child_rep => walk_rep(child_rep))
    return check_target_rep_names
}
// 引数のrep.nameから{type: "", device: "", time: ""}なオブジェクトを作ります。
// rep.nameがdvnf形式ではない場合は、{type: rep.name, device: 'なし', time: ''}が作成されます。
function rep_to_struct(rep: RepStructElementData): { type: string, device: string, time: string } {
    const spl = rep.rep_name.split('_')
    if (spl.length !== 3) {
        return {
            type: rep.rep_name,
            device: 'なし',
            time: ''
        }
    }
    return {
        type: spl[0],
        device: spl[1],
        time: spl[2]
    }
}

async function clicked_reps(e: MouseEvent, items: Array<string>, is_checked: CheckState): Promise<void> {
    return update_check_reps(items, is_checked, true)
}

async function update_reps(items: Array<string>, is_checked: CheckState): Promise<void> {
    return update_check_reps(items, is_checked, false)
}

async function clicked_devices(e: MouseEvent, items: Array<string>, is_checked: CheckState): Promise<void> {
    return update_check_devices(items, is_checked, true)
}

async function update_devices(items: Array<string>, is_checked: CheckState): Promise<void> {
    return update_check_devices(items, is_checked, false)
}

async function clicked_rep_types(e: MouseEvent, items: Array<string>, is_checked: CheckState): Promise<void> {
    return update_check_rep_types(items, is_checked, true)
}

async function update_rep_types(items: Array<string>, is_checked: CheckState): Promise<void> {
    return update_check_rep_types(items, is_checked, false)
}

async function update_check_reps(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean): Promise<void> {
    if (pre_uncheck_all) {
        let f = (_struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            struct.is_checked = false
            struct.indeterminate = false
            if (struct.children) {
                struct.children.forEach(child => f(child))
            }
        }
        f = func
        f(cloned_application_config.value.parsed_rep_struct)
    }

    for (let i = 0; i < items.length; i++) {
        const key_name = items[i]
        let f = (_struct: RepStructElementData) => { }
        let func = (struct: RepStructElementData) => {
            if (struct.key === key_name) {
                switch (is_checked) {
                    case CheckState.checked:
                        struct.is_checked = true
                        struct.indeterminate = false
                        break
                    case CheckState.unchecked:
                        struct.is_checked = false
                        struct.indeterminate = false
                        break
                    case CheckState.indeterminate:
                        struct.is_checked = false
                        struct.indeterminate = true
                        break
                }
            }
            if (struct.children) {
                struct.children.forEach(child => f(child))
            }
        }
        f = func
        f(cloned_application_config.value.parsed_rep_struct)
    }
    const reps = foldable_struct_reps.value?.get_selected_items()
    if (reps && !deepEquals(reps, old_cloned_query.value?.reps)) {
        emits('request_update_checked_reps', reps, true)
    }
    foldable_struct_reps.value?.update_check()
}

async function update_check_devices(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean): Promise<void> {
    if (pre_uncheck_all) {
        let f = (_struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            struct.is_checked = false
            struct.indeterminate = false
            if (struct.children) {
                struct.children.forEach(child => f(child))
            }
        }
        f = func
        f(cloned_application_config.value.parsed_device_struct)
    }

    for (let i = 0; i < items.length; i++) {
        const key_name = items[i]
        let f = (_struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            if (struct.key === key_name) {
                switch (is_checked) {
                    case CheckState.checked:
                        struct.is_checked = true
                        struct.indeterminate = false
                        break
                    case CheckState.unchecked:
                        struct.is_checked = false
                        struct.indeterminate = false
                        break
                    case CheckState.indeterminate:
                        struct.is_checked = false
                        struct.indeterminate = true
                        break
                }
            }
            if (struct.children) {
                struct.children.forEach(child => f(child))
            }
        }
        f = func
        f(cloned_application_config.value.parsed_device_struct)
    }

    const devices = foldable_struct_devices.value?.get_selected_items()
    if (devices && !deepEquals(devices, old_cloned_query.value?.devices_in_sidebar)) {
        emits('request_update_checked_devices', devices, true)
    }
    if (!loading.value) {
        const reps = calc_reps_by_types_and_devices()
        if (reps && !deepEquals(reps, old_cloned_query.value?.reps)) {
            update_check_reps(reps, CheckState.checked, true)
        }
    }
    foldable_struct_devices.value?.update_check()
}

async function update_check_rep_types(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean): Promise<void> {
    if (pre_uncheck_all) {
        let f = (_struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            struct.is_checked = false
            struct.indeterminate = false
            if (struct.children) {
                struct.children.forEach(child => f(child))
            }
        }
        f = func
        f(cloned_application_config.value.parsed_rep_type_struct)
    }

    for (let i = 0; i < items.length; i++) {
        const key_name = items[i]
        let f = (_struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            if (struct.key === key_name) {
                switch (is_checked) {
                    case CheckState.checked:
                        struct.is_checked = true
                        struct.indeterminate = false
                        break
                    case CheckState.unchecked:
                        struct.is_checked = false
                        struct.indeterminate = false
                        break
                    case CheckState.indeterminate:
                        struct.is_checked = false
                        struct.indeterminate = true
                        break
                }
            }
            if (struct.children) {
                struct.children.forEach(child => f(child))
            }
        }
        f = func
        f(cloned_application_config.value.parsed_rep_type_struct)
    }

    const rep_types = foldable_struct_rep_types.value?.get_selected_items()
    if (rep_types && !deepEquals(rep_types, old_cloned_query.value?.rep_types)) {
        emits('request_update_checked_rep_types', rep_types, true)
    }
    if (!loading.value) {
        const reps = calc_reps_by_types_and_devices()
        if (reps && !deepEquals(reps, old_cloned_query.value?.reps)) {
            update_check_reps(reps, CheckState.checked, true)
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

</script>
<style scoped></style>
