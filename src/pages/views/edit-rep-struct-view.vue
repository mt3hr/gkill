<template>
    <v-card>
        <v-card-title>
            記録保管場所構造
        </v-card-title>
        <FoldableStruct :application_config="application_config" :gkill_api="gkill_api" :folder_name="'記録保管場所'"
            :is_open="true" :struct_obj="cloned_application_config.parsed_rep_struct" :is_editable="true"
            :is_root="true" :is_show_checkbox="false"
            @dblclicked_item="(e: MouseEvent, id: string | null) => { if (id) show_edit_rep_struct_dialog(id) }"
            @contextmenu_item="show_rep_contextmenu" ref="foldable_struct" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="show_add_new_rep_struct_element_dialog">記録保管場所追加</v-btn>
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="apply">適用</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="emits('requested_close_dialog')">キャンセル</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AddNewRepStructElementDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :is_open="true" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_add_rep_struct_element="add_rep_struct_element" ref="add_new_rep_struct_element_dialog" />
        <EditRepStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_rep_struct="update_rep_struct" ref="edit_rep_struct_element_dialog" />
        <RepStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="rep_struct_context_menu"
            @requested_edit_rep="(id) => show_edit_rep_struct_dialog(id)"
            @requested_delete_rep="(id) => show_confirm_delete_rep_struct_dialog(id)" />
        <ConfirmDeleteRepStructDialog ref="confirm_delete_rep_struct_dialog" :application_config="application_config"
            :gkill_api="gkill_api" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_rep="(id) => delete_rep_struct(id)" />
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref, watch } from 'vue'
import type { EditRepStructViewEmits } from './edit-rep-struct-view-emits'
import type { EditRepStructViewProps } from './edit-rep-struct-view-props'
import AddNewRepStructElementDialog from '../dialogs/add-new-rep-struct-element-dialog.vue'
import EditRepStructElementDialog from '../dialogs/edit-rep-struct-element-dialog.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import FoldableStruct from './foldable-struct.vue'
import { RepStruct } from '@/classes/datas/config/rep-struct'
import type { FoldableStructModel } from './foldable-struct-model'
import { UpdateRepStructRequest } from '@/classes/api/req_res/update-rep-struct-request'
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import RepStructContextMenu from './rep-struct-context-menu.vue'
import ConfirmDeleteRepStructDialog from '../dialogs/confirm-delete-rep-struct-dialog.vue'

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null);
const edit_rep_struct_element_dialog = ref<InstanceType<typeof EditRepStructElementDialog> | null>(null);
const add_new_rep_struct_element_dialog = ref<InstanceType<typeof AddNewRepStructElementDialog> | null>(null);
const rep_struct_context_menu = ref<InstanceType<typeof RepStructContextMenu> | null>(null);
const confirm_delete_rep_struct_dialog = ref<InstanceType<typeof ConfirmDeleteRepStructDialog> | null>(null);

const props = defineProps<EditRepStructViewProps>()
const emits = defineEmits<EditRepStructViewEmits>()
defineExpose({ reload_cloned_application_config })

watch(() => props.application_config, () => reload_cloned_application_config())

const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

cloned_application_config.value.parse_rep_struct()

async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
    await cloned_application_config.value.append_not_found_reps()
    await cloned_application_config.value.parse_rep_struct()
}

function show_rep_contextmenu(e: MouseEvent, id: string | null): void {
    if (id) {
        rep_struct_context_menu.value?.show(e, id)
    }
}

function show_edit_rep_struct_dialog(id: string): void {
    if (!foldable_struct.value) {
        return
    }
    let target_struct_object: RepStruct | null = null

    for (let i = 0; i < cloned_application_config.value.rep_struct.length; i++) {
        const rep_struct = cloned_application_config.value.rep_struct[i]
        if (rep_struct.id === id) {
            target_struct_object = rep_struct
        }
    }

    if (!target_struct_object) {
        return
    }

    edit_rep_struct_element_dialog.value?.show(target_struct_object)
}

function update_rep_struct(rep_struct_obj: RepStruct): void {
    for (let i = 0; i < cloned_application_config.value.rep_struct.length; i++) {
        if (cloned_application_config.value.rep_struct[i].id === rep_struct_obj.id) {
            cloned_application_config.value.rep_struct.splice(i, 1, rep_struct_obj)
            break
        }
    }
    cloned_application_config.value.parse_rep_struct()
    if (cloned_application_config.value.parsed_rep_struct.children) {
        update_seq(cloned_application_config.value.parsed_rep_struct.children)
    }
}

function update_seq(_rep_struct: Array<FoldableStructModel>): void {
    const exist_ids = new Array<string>()

    // 並び順再決定
    let f = (_struct: FoldableStructModel, _parent: FoldableStructModel, _seq: number) => { }
    let func = (struct: FoldableStructModel, parent: FoldableStructModel, seq: number) => {
        if (struct.id) {
            exist_ids.push(struct.id)
        }
        for (let i = 0; i < cloned_application_config.value.rep_struct.length; i++) {
            if (struct.id === cloned_application_config.value.rep_struct[i].id) {
                cloned_application_config.value.rep_struct[i].seq = seq
                cloned_application_config.value.rep_struct[i].parent_folder_id = parent.id
            }
        }
        if (struct.children) {
            for (let i = 0; i < struct.children.length; i++) {
                f(struct.children[i], struct, i)
            }
        }
    }
    f = func
    if (cloned_application_config.value.parsed_rep_struct.children) {
        for (let i = 0; i < cloned_application_config.value.parsed_rep_struct.children.length; i++) {
            f(cloned_application_config.value.parsed_rep_struct.children[i], cloned_application_config.value.parsed_rep_struct, i)
        }
    }

    // 存在しないものを消す
    for (let i = 0; i < cloned_application_config.value.rep_struct.length; i++) {
        let exist = false
        for (let j = 0; j < exist_ids.length; j++) {
            if (cloned_application_config.value.rep_struct[i].id === exist_ids[j]) {
                exist = true
            }
        }
        if (!exist) {
            cloned_application_config.value.rep_struct.splice(i, 1)
        }
    }
}

async function apply(): Promise<void> {
    const rep_struct = foldable_struct.value?.get_foldable_struct()
    if (!rep_struct) {
        return
    }
    update_seq(rep_struct)

    // 更新する
    const req = new UpdateRepStructRequest()
    req.rep_struct = cloned_application_config.value.rep_struct
    const res = await props.gkill_api.update_rep_struct(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('requested_close_dialog')
}
function show_add_new_rep_struct_element_dialog(): void {
    add_new_rep_struct_element_dialog.value?.show()
}
async function add_rep_struct_element(rep_struct_element: RepStructElementData): Promise<void> {
    const req = new GetGkillInfoRequest()
    const res = await props.gkill_api.get_gkill_info(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    const rep_struct = new RepStruct()
    rep_struct.id = rep_struct_element.id
    rep_struct.user_id = res.user_id
    rep_struct.device = res.device
    rep_struct.check_when_inited = rep_struct_element.check_when_inited
    rep_struct.ignore_check_rep_rykv = rep_struct_element.ignore_check_rep_rykv
    rep_struct.parent_folder_id = null
    rep_struct.seq = cloned_application_config.value.parsed_rep_struct.children ? cloned_application_config.value.parsed_rep_struct.children.length : 0
    rep_struct.rep_name = rep_struct_element.rep_name

    cloned_application_config.value.rep_struct.push(rep_struct)
    cloned_application_config.value.parsed_rep_struct.children?.push(rep_struct_element)
}
function show_confirm_delete_rep_struct_dialog(id: string): void {
    if (!foldable_struct.value) {
        return
    }
    let target_struct_object: RepStruct | null = null

    for (let i = 0; i < cloned_application_config.value.rep_struct.length; i++) {
        const rep_struct = cloned_application_config.value.rep_struct[i]
        if (rep_struct.id === id) {
            target_struct_object = rep_struct
        }
    }

    if (!target_struct_object) {
        return
    }
    confirm_delete_rep_struct_dialog.value?.show(target_struct_object)
}
function delete_rep_struct(id: string): void {
    for (let i = 0; i < cloned_application_config.value.rep_struct.length; i++) {
        if (cloned_application_config.value.rep_struct[i].id === id) {
            cloned_application_config.value.rep_struct.splice(i, 1)
            break
        }
    }
    foldable_struct.value?.delete_struct(id)
}
</script>