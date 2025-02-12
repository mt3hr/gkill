<template>
    <v-card>
        <v-card-title>
            KFTLテンプレート構造
        </v-card-title>
        <FoldableStruct :application_config="application_config" :gkill_api="gkill_api" :folder_name="'KFTLテンプレート'"
            :is_open="true" :struct_obj="cloned_application_config.parsed_kftl_template" :is_editable="true"
            :is_root="true" :is_show_checkbox="false" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @dblclicked_item="(e: MouseEvent, id: string | null) => { if (id) show_edit_kftl_template_struct_dialog(id) }"
            @contextmenu_item="show_kftl_template_contextmenu" ref="foldable_struct" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="show_add_new_kftl_template_struct_element_dialog">KFTLテンプレート追加</v-btn>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="show_add_new_folder_dialog">フォルダ追加</v-btn>
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
        <AddNewFoloderDialog :application_config="application_config" :gkill_api="gkill_api"
            @requested_add_new_folder="add_folder_struct_element"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="add_new_folder_dialog" />
        <AddNewKFTLTemplateStructElementDialog :application_config="application_config" :folder_name="''"
            :gkill_api="gkill_api" :is_open="true" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_add_kftl_template_struct_element="add_kftl_template_struct_element"
            ref="add_new_kftl_template_struct_element_dialog" />
        <EditKFTLTemplateStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_kftl_template_struct="update_kftl_template_struct"
            ref="edit_kftl_template_struct_element_dialog" />
        <KFTLTemplateStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            ref="kftl_template_struct_context_menu"
            @requested_edit_kftl_template="(id) => show_edit_kftl_template_struct_dialog(id)"
            @requested_delete_kftl_template="(id) => show_confirm_delete_kftl_template_struct_dialog(id)" />
        <ConfirmDeleteKFTLTemplateStructDialog ref="confirm_delete_kftl_template_struct_dialog"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_kftl_template="(id) => delete_kftl_template_struct(id)" />
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref, watch } from 'vue'
import type { EditKFTLTemplateStructViewEmits } from './edit-kftl-template-struct-view-emits.ts'
import type { EditKFTLTemplateStructViewProps } from './edit-kftl-template-struct-view-props.ts'
import AddNewKFTLTemplateStructElementDialog from '../dialogs/add-new-kftl-template-struct-element-dialog.vue'
import EditKFTLTemplateStructElementDialog from '../dialogs/edit-kftl-template-struct-element-dialog.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import FoldableStruct from './foldable-struct.vue'
import { KFTLTemplateStruct } from '@/classes/datas/config/kftl-template-struct'
import type { FoldableStructModel } from './foldable-struct-model'
import AddNewFoloderDialog from '../dialogs/add-new-foloder-dialog.vue'
import { KFTLTemplateStructElementData } from '@/classes/datas/config/kftl-template-struct-element-data'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import KFTLTemplateStructContextMenu from './kftl-template-struct-context-menu.vue'
import ConfirmDeleteKFTLTemplateStructDialog from '../dialogs/confirm-delete-kftl-template-struct-dialog.vue'
import { UpdateKFTLTemplateRequest } from '@/classes/api/req_res/update-kftl-template-request.js'

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null);
const edit_kftl_template_struct_element_dialog = ref<InstanceType<typeof EditKFTLTemplateStructElementDialog> | null>(null);
const add_new_folder_dialog = ref<InstanceType<typeof AddNewFoloderDialog> | null>(null);
const add_new_kftl_template_struct_element_dialog = ref<InstanceType<typeof AddNewKFTLTemplateStructElementDialog> | null>(null);
const kftl_template_struct_context_menu = ref<InstanceType<typeof KFTLTemplateStructContextMenu> | null>(null);
const confirm_delete_kftl_template_struct_dialog = ref<InstanceType<typeof ConfirmDeleteKFTLTemplateStructDialog> | null>(null);

const props = defineProps<EditKFTLTemplateStructViewProps>()
const emits = defineEmits<EditKFTLTemplateStructViewEmits>()
defineExpose({ reload_cloned_application_config })

watch(() => props.application_config, () => reload_cloned_application_config())

const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

cloned_application_config.value.parse_kftl_template_struct()

async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
    cloned_application_config.value.parse_kftl_template_struct()
}

function show_kftl_template_contextmenu(e: MouseEvent, id: string | null): void {
    if (id) {
        kftl_template_struct_context_menu.value?.show(e, id)
    }
}

function show_edit_kftl_template_struct_dialog(id: string): void {
    if (!foldable_struct.value) {
        return
    }
    let target_struct_object: KFTLTemplateStruct | null = null

    for (let i = 0; i < cloned_application_config.value.kftl_template_struct.length; i++) {
        const kftl_template_struct = cloned_application_config.value.kftl_template_struct[i]
        if (kftl_template_struct.id === id) {
            target_struct_object = kftl_template_struct
        }
    }

    if (!target_struct_object) {
        return
    }

    edit_kftl_template_struct_element_dialog.value?.show(target_struct_object)
}

function update_kftl_template_struct(kftl_template_struct_obj: KFTLTemplateStruct): void {
    for (let i = 0; i < cloned_application_config.value.kftl_template_struct.length; i++) {
        if (cloned_application_config.value.kftl_template_struct[i].id === kftl_template_struct_obj.id) {
            cloned_application_config.value.kftl_template_struct.splice(i, 1, kftl_template_struct_obj)
            break
        }
    }
    cloned_application_config.value.parse_kftl_template_struct()
    if (cloned_application_config.value.parsed_kftl_template.children) {
        update_seq(cloned_application_config.value.parsed_kftl_template.children)
    }
}

function update_seq(_kftl_template_struct: Array<FoldableStructModel>): void {
    const exist_ids = new Array<string>()

    // 並び順再決定
    let f = (_struct: FoldableStructModel, _parent: FoldableStructModel, _seq: number) => { }
    let func = (struct: FoldableStructModel, parent: FoldableStructModel, seq: number) => {
        if(struct.id) {
            exist_ids.push(struct.id)
        }
        for (let i = 0; i < cloned_application_config.value.kftl_template_struct.length; i++) {
            if (struct.id === cloned_application_config.value.kftl_template_struct[i].id) {
                cloned_application_config.value.kftl_template_struct[i].seq = seq
                cloned_application_config.value.kftl_template_struct[i].parent_folder_id = parent.id
            }
        }
        if (struct.children) {
            for (let i = 0; i < struct.children.length; i++) {
                f(struct.children[i], struct, i)
            }
        }
    }
    f = func
    if (cloned_application_config.value.parsed_kftl_template.children) {
        for (let i = 0; i < cloned_application_config.value.parsed_kftl_template.children?.length; i++) {
            f(cloned_application_config.value.parsed_kftl_template.children[i], cloned_application_config.value.parsed_kftl_template, i)
        }
    }

    // 存在しないものを消す
    for (let i = 0; i < cloned_application_config.value.kftl_template_struct.length; i++) {
        let exist = false
        for (let j = 0; j < exist_ids.length; j++) {
            if (cloned_application_config.value.kftl_template_struct[i].id === exist_ids[j]) {
                exist = true
            }
        }
        if (!exist) {
            cloned_application_config.value.kftl_template_struct.splice(i, 1)
            i--
        }
    }

}

async function apply(): Promise<void> {
    const kftl_template_struct = foldable_struct.value?.get_foldable_struct()
    if (!kftl_template_struct) {
        return
    }

    update_seq(kftl_template_struct)

    // 更新する
    const req = new UpdateKFTLTemplateRequest()
    req.kftl_templates = cloned_application_config.value.kftl_template_struct
    const res = await props.gkill_api.update_kftl_template(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('requested_close_dialog')
}
function show_add_new_kftl_template_struct_element_dialog(): void {
    add_new_kftl_template_struct_element_dialog.value?.show()
}
function show_add_new_folder_dialog(): void {
    add_new_folder_dialog.value?.show()
}
async function add_folder_struct_element(folder_struct_element: FolderStructElementData): Promise<void> {
    const req = new GetGkillInfoRequest()
    const res = await props.gkill_api.get_gkill_info(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    const kftl_template_struct = new KFTLTemplateStruct()
    kftl_template_struct.id = folder_struct_element.id
    kftl_template_struct.user_id = res.user_id
    kftl_template_struct.device = res.device
    kftl_template_struct.title = folder_struct_element.folder_name
    kftl_template_struct.template = null
    kftl_template_struct.parent_folder_id = null
    kftl_template_struct.seq = cloned_application_config.value.parsed_kftl_template.children ? cloned_application_config.value.parsed_kftl_template.children.length : 0

    const kftl_template_struct_element = new KFTLTemplateStructElementData()
    kftl_template_struct_element.id = folder_struct_element.id
    kftl_template_struct_element.title = folder_struct_element.folder_name
    kftl_template_struct_element.children = new Array<KFTLTemplateStructElementData>()
    kftl_template_struct_element.key = folder_struct_element.folder_name

    cloned_application_config.value.kftl_template_struct.push(kftl_template_struct)
    cloned_application_config.value.parsed_kftl_template.children?.push(kftl_template_struct_element)
}
async function add_kftl_template_struct_element(kftl_template_struct_element: KFTLTemplateStructElementData): Promise<void> {
    const req = new GetGkillInfoRequest()
    const res = await props.gkill_api.get_gkill_info(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    const kftl_template_struct = new KFTLTemplateStruct()
    kftl_template_struct.id = kftl_template_struct_element.id ? kftl_template_struct_element.id : ""
    kftl_template_struct.user_id = res.user_id
    kftl_template_struct.device = res.device
    kftl_template_struct.title = kftl_template_struct_element.title
    kftl_template_struct.template = kftl_template_struct_element.template
    kftl_template_struct.parent_folder_id = null
    kftl_template_struct.seq = cloned_application_config.value.parsed_kftl_template.children ? cloned_application_config.value.parsed_kftl_template.children.length : 0

    cloned_application_config.value.kftl_template_struct.push(kftl_template_struct)
    cloned_application_config.value.parsed_kftl_template.children?.push(kftl_template_struct_element)
}
function show_confirm_delete_kftl_template_struct_dialog(id: string): void {
    if (!foldable_struct.value) {
        return
    }
    let target_struct_object: KFTLTemplateStruct | null = null

    for (let i = 0; i < cloned_application_config.value.kftl_template_struct.length; i++) {
        const kftl_template_struct = cloned_application_config.value.kftl_template_struct[i]
        if (kftl_template_struct.id === id) {
            target_struct_object = kftl_template_struct
        }
    }

    if (!target_struct_object) {
        return
    }
    confirm_delete_kftl_template_struct_dialog.value?.show(target_struct_object)
}
function delete_kftl_template_struct(id: string): void {
    for (let i = 0; i < cloned_application_config.value.kftl_template_struct.length; i++) {
        if (cloned_application_config.value.kftl_template_struct[i].id === id) {
            cloned_application_config.value.kftl_template_struct.splice(i, 1)
            break
        }
    }
    foldable_struct.value?.delete_struct(id)
}
</script>
