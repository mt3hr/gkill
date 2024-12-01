<template>
    <div class="replist">
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox readonly v-model="use_rep" label="記録保管場所" hide-details />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0">
                <v-btn @click="emits('request_clear_rep_query')" hide-details>クリア</v-btn>
            </v-col>
        </v-row>
        <v-tabs v-show="use_rep" v-model="tab">
            <v-tab key="summary">Summary</v-tab>
            <v-tab key="detail">Detail</v-tab>
        </v-tabs>
        <v-window v-model="tab">
            <v-window-item key="summary" eager="true">
                <h2>Devices</h2>
                <table class="devicelist">
                    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                        :is_editable="false" :is_root="true" :is_show_checkbox="true" :is_open="true" :query="query"
                        :struct_obj="cloned_application_config.parsed_device_struct"
                        @requested_update_check_state="update_devices"
                        @received_errors="(errors) => emits('received_errors', errors)"
                        @received_messages="(messages) => emits('received_messages', messages)"
                        @clicked_items="clicked_devices" ref="foldable_struct_rep_devices" />
                </table>
                <h2>Types</h2>
                <table class="typelist">
                    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                        :is_editable="false" :is_root="true" :is_show_checkbox="true" :is_open="true" :query="query"
                        :struct_obj="cloned_application_config.parsed_rep_type_struct"
                        @requested_update_check_state="update_rep_types"
                        @received_errors="(errors) => emits('received_errors', errors)"
                        @received_messages="(messages) => emits('received_messages', messages)"
                        @clicked_items="clicked_rep_types" ref="folfable_struct_rep_types" />
                </table>
            </v-window-item>
            <v-window-item key="detail" eager="true">
                <h2>Reps</h2>
                <table>
                    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                        :is_editable="false" :is_root="true" :is_show_checkbox="true" :is_open="true" :query="query"
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
import { type Ref, ref, watch, nextTick, onMounted } from 'vue'
import FoldableStruct from './foldable-struct.vue'
import type { RepQueryEmits } from './rep-query-emits'
import type { RepQueryProps } from './rep-query-props'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import type { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data'
import type { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'
import { CheckState } from './check-state'
import type { FoldableStructModel } from './foldable-struct-model'

const foldable_struct_reps = ref<InstanceType<typeof FoldableStruct> | null>(null)
const foldable_struct_devices = ref<InstanceType<typeof FoldableStruct> | null>(null)
const foldable_struct_rep_types = ref<InstanceType<typeof FoldableStruct> | null>(null)

const props = defineProps<RepQueryProps>()
const emits = defineEmits<RepQueryEmits>()
defineExpose({ get_checked_reps })

const cloned_query: Ref<FindKyouQuery> = ref(await props.query.clone())
const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

watch(() => props.application_config, async () => {
    cloned_application_config.value = props.application_config.clone()
    cloned_application_config.value.parse_rep_struct()
    cloned_application_config.value.parse_device_struct()
    cloned_application_config.value.parse_rep_type_struct()

    await calc_reps_by_types_and_devices_promise()
    const checked_items = await foldable_struct_reps.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_reps', checked_items)
    }
})

const tab = ref(2)
const use_rep = ref(true)

const checked_reps: Ref<Array<RepStructElementData>> = ref(new Array<RepStructElementData>())
const checked_devices: Ref<Array<DeviceStructElementData>> = ref(new Array<DeviceStructElementData>())
const checked_rep_types: Ref<Array<RepTypeStructElementData>> = ref(new Array<RepTypeStructElementData>())

function get_all_reps() {
    let reps_objects: Array<RepStructElementData> = []
    let root_rep = cloned_application_config.value.parsed_rep_struct
    let f = (struct_obj: RepStructElementData) => { }
    let func = (struct_obj: RepStructElementData) => {
        reps_objects.push(struct_obj)
        if (struct_obj.children) {
            struct_obj.children.forEach((child) => f(child))
        }
    }
    f = func
    if (root_rep) {
        f(root_rep)
    }
    return reps_objects
}
function get_all_devices() {
    let devices_objects: Array<DeviceStructElementData> = []
    let root_device = cloned_application_config.value.parsed_device_struct
    let f = (struct_obj: DeviceStructElementData) => { }
    let func = (struct_obj: DeviceStructElementData) => {
        devices_objects.push(struct_obj)
        if (struct_obj.children) {
            struct_obj.children.forEach((child) => f(child))
        }
    }
    f = func
    if (root_device) {
        f(root_device)
    }
    return devices_objects
}
function get_all_rep_types() {
    let rep_types_objects: Array<RepTypeStructElementData> = []
    let root_rep_type = cloned_application_config.value.parsed_rep_type_struct
    let f = (struct_obj: RepTypeStructElementData) => { }
    let func = (struct_obj: RepTypeStructElementData) => {
        rep_types_objects.push(struct_obj)
        if (struct_obj.children) {
            struct_obj.children.forEach((child) => f(child))
        }
    }
    f = func
    if (root_rep_type) {
        f(root_rep_type)
    }
    return rep_types_objects
}


// checked_repsが更新されたら、repsオブジェクトを走り回ってchecked_repsのrepにチェックを入れていく
watch(() => checked_reps, async () => {
    const reps = get_all_reps()
    for (let i = 0; i < reps.length; i++) {
        const rep = reps[i]
        rep.is_checked = false
        for (let j = 0; j < checked_reps.value.length; j++) {
            if (rep.key === checked_reps.value[j].key) {
                rep.is_checked = true
                break
            }
        }
    }
})
// checked_devicesが更新されたら、repsオブジェクトを走り回ってchecked_devicesにマッチするrepにチェックを入れていく
watch(() => checked_devices, async () => {
    const devices = get_all_devices()
    for (let i = 0; i < devices.length; i++) {
        let device = devices[i]
        device.is_checked = false
        for (let j = 0; j < checked_devices.value.length; j++) {
            if (device.key === checked_devices.value[j].key) {
                device.is_checked = true
                break
            }
        }
    }
    await calc_reps_by_types_and_devices_promise()
})
// checked_typesが更新されたら、repsオブジェクトを走り回ってchecked_typesにマッチするrepにチェックを入れていく
watch(() => checked_rep_types, async () => {
    const rep_types = get_all_rep_types()
    for (let i = 0; i < rep_types.length; i++) {
        let rep_type = rep_types[i]
        rep_type.is_checked = false
        for (let j = 0; j < checked_rep_types.value.length; j++) {
            if (rep_type.key === checked_rep_types.value[j].key) {
                rep_type.is_checked = true
                break
            }
        }
    }
    await calc_reps_by_types_and_devices_promise()
})

// 現在チェックされているdevices, typesに該当するrep.nameを抽出してthis.repsを更新し、emitします。
function calc_reps_by_types_and_devices_promise(): void {
    const reps = get_all_reps()
    const rep_types = get_all_rep_types()
    const devices = get_all_devices()
    reps.forEach(rep => {
        rep.is_checked = false
        const rep_struct = rep_to_struct(rep)

        let type_is_match = false
        let device_is_match = false
        rep_types.forEach(rep_type => {
            rep_type.indeterminate = false
            if (rep_type.is_checked && rep_type.key === rep_struct.type) {
                type_is_match = true
            }
        })
        devices.forEach(device => {
            device.indeterminate = false
            if (device.is_checked && device.key === rep_struct.device) {
                device_is_match = true
            }
        })

        if (type_is_match && device_is_match) {
            rep.is_checked = true
        }
    })
}
// 引数のrep.nameから{type: "", device: "", time: ""}なオブジェクトを作ります。
// rep.nameがdvnf形式ではない場合は、{type: rep.name, device: 'なし', time: ''}が作成されます。
function rep_to_struct(rep: RepStructElementData): { type: string, device: string, time: string } {
    const spl = rep.key.split('_', 3)
    if (spl.length !== 3) {
        return {
            type: rep.key,
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
    update_check_reps(items, is_checked, true)
}

async function update_reps(items: Array<string>, is_checked: CheckState): Promise<void> {
    update_check_reps(items, is_checked, false)
}

async function clicked_devices(e: MouseEvent, items: Array<string>, is_checked: CheckState): Promise<void> {
    update_check_devices(items, is_checked, true)
}

async function update_devices(items: Array<string>, is_checked: CheckState): Promise<void> {
    update_check_devices(items, is_checked, false)
}

async function clicked_rep_types(e: MouseEvent, items: Array<string>, is_checked: CheckState): Promise<void> {
    update_check_rep_types(items, is_checked, true)
}

async function update_rep_types(items: Array<string>, is_checked: CheckState): Promise<void> {
    update_check_rep_types(items, is_checked, false)
}

async function update_check_reps(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean): Promise<void> {
    const reps = get_all_reps()
    if (pre_uncheck_all) {
        let f = (struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            struct.is_checked = false
            struct.indeterminate = false
            if (struct.children) {
                struct.children.forEach(child => {
                    f(child)
                })
            }
        }
        f = func
        reps.forEach((rep) => f(rep))
    }

    for (let i = 0; i < items.length; i++) {
        const key_name = items[i]
        let f = (struct: FoldableStructModel) => { }
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
                struct.children.forEach(child => {
                    f(child)
                })
            }
        }
        f = func
        reps.forEach((rep) => f(rep))
    }

    const checked_items = await foldable_struct_reps.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_reps', checked_items)
    }
}

async function update_check_devices(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean): Promise<void> {
    const devices = get_all_devices()
    if (pre_uncheck_all) {
        let f = (struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            struct.is_checked = false
            struct.indeterminate = false
            if (struct.children) {
                struct.children.forEach(child => {
                    f(child)
                })
            }
        }
        f = func
        devices.forEach(device => f(device))
    }

    for (let i = 0; i < items.length; i++) {
        const key_name = items[i]
        let f = (struct: FoldableStructModel) => { }
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
                struct.children.forEach(child => {
                    f(child)
                })
            }
        }
        f = func
        devices.forEach(device => f(device))
    }

    await calc_reps_by_types_and_devices_promise()
    const checked_items = await foldable_struct_reps.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_reps', checked_items)
    }
}

async function update_check_rep_types(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean): Promise<void> {
    const rep_types = get_all_rep_types()
    if (pre_uncheck_all) {
        let f = (struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            struct.is_checked = false
            struct.indeterminate = false
            if (struct.children) {
                struct.children.forEach(child => {
                    f(child)
                })
            }
        }
        f = func
        rep_types.forEach(rep_type => f(rep_type))
    }

    for (let i = 0; i < items.length; i++) {
        const key_name = items[i]
        let f = (struct: FoldableStructModel) => { }
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
                struct.children.forEach(child => {
                    f(child)
                })
            }
        }
        f = func
        rep_types.forEach(rep_type => f(rep_type))
    }

    await calc_reps_by_types_and_devices_promise()
    const checked_items = await foldable_struct_reps.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_reps', checked_items)
    }
}

function get_checked_reps(): Array<string> {
    const reps = foldable_struct_reps.value?.get_selected_items()
    if (reps) {
        return reps
    }
    return new Array<string>()
}

</script>
<style scoped></style>
