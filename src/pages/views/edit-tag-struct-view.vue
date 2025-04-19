<template>
    <v-card>
        <v-card-title>
            {{ $t("TAG_STRUCT_TITLE") }}
        </v-card-title>
        <div class="tag_struct_root">
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api" :folder_name="$t('TAG_TITLE')"
                :is_open="true" :struct_obj="cloned_application_config.parsed_tag_struct" :is_editable="true"
                :is_root="true" :is_show_checkbox="false"
                @dblclicked_item="(e: MouseEvent, id: string | null) => { if (id) show_edit_tag_struct_dialog(id) }"
                @contextmenu_item="show_tag_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_tag_struct_element_dialog">{{ $t("ADD_TAG_TITLE") }}</v-btn>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_folder_dialog">{{ $t("ADD_FOLDER_TITLE") }}</v-btn>
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="apply" color="primary">{{ $t("APPLY_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ $t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AddNewFoloderDialog :application_config="application_config" :gkill_api="gkill_api"
            @requested_add_new_folder="add_folder_struct_element"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="add_new_folder_dialog" />
        <AddNewTagStructElementDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :is_open="true" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_add_tag_struct_element="add_tag_struct_element" ref="add_new_tag_struct_element_dialog" />
        <EditTagStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_tag_struct="update_tag_struct" ref="edit_tag_struct_element_dialog" />
        <TagStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="tag_struct_context_menu"
            @requested_edit_tag="(id) => show_edit_tag_struct_dialog(id)"
            @requested_delete_tag="(id) => show_confirm_delete_tag_struct_dialog(id)" />
        <ConfirmDeleteTagStructDialog ref="confirm_delete_tag_struct_dialog" :application_config="application_config"
            :gkill_api="gkill_api" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_tag="(id) => delete_tag_struct(id)" />
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref, watch } from 'vue'
import type { EditTagStructViewEmits } from './edit-tag-struct-view-emits'
import type { EditTagStructViewProps } from './edit-tag-struct-view-props'
import AddNewTagStructElementDialog from '../dialogs/add-new-tag-struct-element-dialog.vue'
import EditTagStructElementDialog from '../dialogs/edit-tag-struct-element-dialog.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import FoldableStruct from './foldable-struct.vue'
import { TagStruct } from '@/classes/datas/config/tag-struct'
import type { FoldableStructModel } from './foldable-struct-model'
import { UpdateTagStructRequest } from '@/classes/api/req_res/update-tag-struct-request'
import AddNewFoloderDialog from '../dialogs/add-new-foloder-dialog.vue'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import TagStructContextMenu from './tag-struct-context-menu.vue'
import ConfirmDeleteTagStructDialog from '../dialogs/confirm-delete-tag-struct-dialog.vue'

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null);
const edit_tag_struct_element_dialog = ref<InstanceType<typeof EditTagStructElementDialog> | null>(null);
const add_new_folder_dialog = ref<InstanceType<typeof AddNewFoloderDialog> | null>(null);
const add_new_tag_struct_element_dialog = ref<InstanceType<typeof AddNewTagStructElementDialog> | null>(null);
const tag_struct_context_menu = ref<InstanceType<typeof TagStructContextMenu> | null>(null);
const confirm_delete_tag_struct_dialog = ref<InstanceType<typeof ConfirmDeleteTagStructDialog> | null>(null);

const props = defineProps<EditTagStructViewProps>()
const emits = defineEmits<EditTagStructViewEmits>()
defineExpose({ reload_cloned_application_config })

watch(() => props.application_config, () => reload_cloned_application_config())

const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

cloned_application_config.value.parse_tag_struct()

async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
    await cloned_application_config.value.append_not_found_tags()
    await cloned_application_config.value.parse_tag_struct()
}

function show_tag_contextmenu(e: MouseEvent, id: string | null): void {
    if (id) {
        tag_struct_context_menu.value?.show(e, id)
    }
}

function show_edit_tag_struct_dialog(id: string): void {
    if (!foldable_struct.value) {
        return
    }
    let target_struct_object: TagStruct | null = null

    for (let i = 0; i < cloned_application_config.value.tag_struct.length; i++) {
        const tag_struct = cloned_application_config.value.tag_struct[i]
        if (tag_struct.id === id) {
            target_struct_object = tag_struct
        }
    }

    if (!target_struct_object) {
        return
    }

    edit_tag_struct_element_dialog.value?.show(target_struct_object)
}

function update_tag_struct(tag_struct_obj: TagStruct): void {
    for (let i = 0; i < cloned_application_config.value.tag_struct.length; i++) {
        if (cloned_application_config.value.tag_struct[i].id === tag_struct_obj.id) {
            cloned_application_config.value.tag_struct.splice(i, 1, tag_struct_obj)
            break
        }
    }
    if (cloned_application_config.value.parsed_tag_struct.children) {
        update_seq(cloned_application_config.value.parsed_tag_struct.children)
    }
}

function update_seq(_tag_struct: Array<FoldableStructModel>): void {
    const exist_ids = new Array<string>()

    // 並び順再決定
    let f = (_struct: FoldableStructModel, _parent: FoldableStructModel, _seq: number) => { }
    let func = (struct: FoldableStructModel, parent: FoldableStructModel, seq: number) => {
        if (struct.id) {
            exist_ids.push(struct.id)
        }

        for (let i = 0; i < cloned_application_config.value.tag_struct.length; i++) {
            if (struct.id === cloned_application_config.value.tag_struct[i].id) {
                cloned_application_config.value.tag_struct[i].seq = seq
                cloned_application_config.value.tag_struct[i].parent_folder_id = parent.id
            }
        }
        if (struct.children) {
            for (let i = 0; i < struct.children.length; i++) {
                f(struct.children[i], struct, i)
            }
        }
    }
    f = func

    if (cloned_application_config.value.parsed_tag_struct.children) {
        for (let i = 0; i < cloned_application_config.value.parsed_tag_struct.children.length; i++) {
            f(cloned_application_config.value.parsed_tag_struct.children[i], cloned_application_config.value.parsed_tag_struct, i)
        }
    }

    // 存在しないものを消す
    for (let i = 0; i < cloned_application_config.value.tag_struct.length; i++) {
        let exist = false
        for (let j = 0; j < exist_ids.length; j++) {
            if (cloned_application_config.value.tag_struct[i].id === exist_ids[j]) {
                exist = true
            }
        }
        if (!exist) {
            cloned_application_config.value.tag_struct.splice(i, 1)
            i--
        }
    }
}

async function apply(): Promise<void> {
    const tag_struct = foldable_struct.value?.get_foldable_struct()
    if (!tag_struct) {
        return
    }
    update_seq(tag_struct)

    // 更新する
    const req = new UpdateTagStructRequest()
    req.tag_struct = cloned_application_config.value.tag_struct
    const res = await props.gkill_api.update_tag_struct(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('requested_close_dialog')
}
function show_add_new_tag_struct_element_dialog(): void {
    add_new_tag_struct_element_dialog.value?.show()
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

    const tag_struct = new TagStruct()
    tag_struct.id = folder_struct_element.id
    tag_struct.user_id = res.user_id
    tag_struct.device = res.device
    tag_struct.check_when_inited = false
    tag_struct.is_force_hide = false
    tag_struct.parent_folder_id = null
    tag_struct.seq = cloned_application_config.value.parsed_tag_struct.children ? cloned_application_config.value.parsed_tag_struct.children.length : 0
    tag_struct.tag_name = folder_struct_element.folder_name

    const tag_struct_element = new TagStructElementData()
    tag_struct_element.id = folder_struct_element.id
    tag_struct_element.check_when_inited = false
    tag_struct_element.is_force_hide = false
    tag_struct_element.tag_name = folder_struct_element.folder_name
    tag_struct_element.children = new Array<TagStructElementData>()
    tag_struct_element.key = folder_struct_element.folder_name

    cloned_application_config.value.tag_struct.push(tag_struct)
    cloned_application_config.value.parsed_tag_struct.children?.push(tag_struct_element)
}
async function add_tag_struct_element(tag_struct_element: TagStructElementData): Promise<void> {
    const req = new GetGkillInfoRequest()
    const res = await props.gkill_api.get_gkill_info(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    const tag_struct = new TagStruct()
    tag_struct.id = tag_struct_element.id ? tag_struct_element.id : ""
    tag_struct.user_id = res.user_id
    tag_struct.device = res.device
    tag_struct.check_when_inited = tag_struct_element.check_when_inited
    tag_struct.is_force_hide = tag_struct_element.is_force_hide
    tag_struct.parent_folder_id = null
    tag_struct.seq = cloned_application_config.value.parsed_tag_struct.children ? cloned_application_config.value.parsed_tag_struct.children.length : 0
    tag_struct.tag_name = tag_struct_element.tag_name

    cloned_application_config.value.tag_struct.push(tag_struct)
    cloned_application_config.value.parsed_tag_struct.children?.push(tag_struct_element)
    await cloned_application_config.value.parse_tag_struct()
}
function show_confirm_delete_tag_struct_dialog(id: string): void {
    if (!foldable_struct.value) {
        return
    }
    let target_struct_object: TagStruct | null = null

    for (let i = 0; i < cloned_application_config.value.tag_struct.length; i++) {
        const tag_struct = cloned_application_config.value.tag_struct[i]
        if (tag_struct.id === id) {
            target_struct_object = tag_struct
        }
    }

    if (!target_struct_object) {
        return
    }
    confirm_delete_tag_struct_dialog.value?.show(target_struct_object)
}
function delete_tag_struct(id: string): void {
    for (let i = 0; i < cloned_application_config.value.tag_struct.length; i++) {
        if (cloned_application_config.value.tag_struct[i].id === id) {
            cloned_application_config.value.tag_struct.splice(i, 1)
            break
        }
    }
    foldable_struct.value?.delete_struct(id)
}
</script>
<style lang="css" scoped>
.tag_struct_root {
    max-height: 80vh;
    overflow-y: scroll;
}
</style>