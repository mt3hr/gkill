<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("TAG_STRUCT_TITLE") }}
        </v-card-title>
        <div class="tag_struct_root">
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api"
                :folder_name="i18n.global.t('TAG_TITLE')" :is_open="true"
                :struct_obj="cloned_application_config.tag_struct" :is_editable="true" :is_root="true"
                :is_show_checkbox="false"
                @dblclicked_item="(e: MouseEvent, id: string | null) => { if (id) show_edit_tag_struct_dialog(id) }"
                @contextmenu_item="show_tag_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_tag_struct_element_dialog">{{
                        i18n.global.t("ADD_TAG_TITLE") }}</v-btn>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_folder_dialog">{{ i18n.global.t("ADD_FOLDER_TITLE")
                        }}</v-btn>
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="apply" color="primary">{{ i18n.global.t("APPLY_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{
                        i18n.global.t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AddNewFoloderDialog :application_config="application_config" :gkill_api="gkill_api"
            @requested_add_new_folder="add_folder_struct_element"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="add_new_folder_dialog" />
        <AddNewTagStructElementDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :is_open="true"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_add_tag_struct_element="add_tag_struct_element" ref="add_new_tag_struct_element_dialog" />
        <EditTagStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_update_tag_struct="update_tag_struct" ref="edit_tag_struct_element_dialog" />
        <TagStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="tag_struct_context_menu"
            @requested_edit_tag="(...id: any[]) => show_edit_tag_struct_dialog(id[0] as string)"
            @requested_delete_tag="(...id: any[]) => show_confirm_delete_tag_struct_dialog(id[0] as string)" />
        <ConfirmDeleteTagStructDialog ref="confirm_delete_tag_struct_dialog" :application_config="application_config"
            :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_delete_tag="(...id: any[]) => delete_tag_struct(id[0] as string)" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditTagStructViewEmits } from './edit-tag-struct-view-emits'
import type { EditTagStructViewProps } from './edit-tag-struct-view-props'
import AddNewTagStructElementDialog from '../dialogs/add-new-tag-struct-element-dialog.vue'
import EditTagStructElementDialog from '../dialogs/edit-tag-struct-element-dialog.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import FoldableStruct from './foldable-struct.vue'
import AddNewFoloderDialog from '../dialogs/add-new-foloder-dialog.vue'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'

import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import TagStructContextMenu from './tag-struct-context-menu.vue'
import ConfirmDeleteTagStructDialog from '../dialogs/confirm-delete-tag-struct-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

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
async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
    await cloned_application_config.value.append_not_found_tags()
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
    let target_struct_object: TagStructElementData | null = null
    let tag_name_walk = (_tag: TagStructElementData): void => { }
    tag_name_walk = (tag: TagStructElementData): void => {
        const tag_children = tag.children
        if (tag.id === id) {
            target_struct_object = tag
        } else if (tag_children) {
            tag_children.forEach(child_tag => {
                if (child_tag) {
                    tag_name_walk(child_tag)
                }
            })
        }
    }
    tag_name_walk(cloned_application_config.value.tag_struct)

    if (!target_struct_object) {
        return
    }

    edit_tag_struct_element_dialog.value?.show(target_struct_object)
}

function update_tag_struct(tag_struct_obj: TagStructElementData): void {
    let tag_name_walk = (_tag: TagStructElementData): boolean => false
    tag_name_walk = (tag: TagStructElementData): boolean => {
        const tag_children = tag.children
        if (tag.id === tag_struct_obj.id) {
            return true
        } else if (tag_children) {
            for (let i = 0; i < tag_children.length; i++) {
                const child_tag = tag_children[i]
                if (child_tag.children) {
                    if (tag_name_walk(child_tag)) {
                        tag_children[i] = tag_struct_obj
                        return false
                    }
                }
            }
        }
        return false
    }
    tag_name_walk(cloned_application_config.value.tag_struct)
}

async function apply(): Promise<void> {
    emits('requested_apply_tag_struct', cloned_application_config.value.tag_struct)
    nextTick(() => emits('requested_close_dialog'))
}
function show_add_new_tag_struct_element_dialog(): void {
    add_new_tag_struct_element_dialog.value?.show()
}
function show_add_new_folder_dialog(): void {
    add_new_folder_dialog.value?.show()
}
async function add_folder_struct_element(folder_struct_element: FolderStructElementData): Promise<void> {
    const tag_struct_element = new TagStructElementData()
    tag_struct_element.id = folder_struct_element.id
    tag_struct_element.is_dir = true
    tag_struct_element.check_when_inited = false
    tag_struct_element.tag_name = folder_struct_element.folder_name
    tag_struct_element.children = new Array<TagStructElementData>()
    tag_struct_element.key = folder_struct_element.folder_name
    cloned_application_config.value.tag_struct.children?.push(tag_struct_element)
}
async function add_tag_struct_element(tag_struct_element: TagStructElementData): Promise<void> {
    cloned_application_config.value.tag_struct.children?.push(tag_struct_element)
}
function show_confirm_delete_tag_struct_dialog(id: string): void {
    let target_struct_object: TagStructElementData | null = null
    let tag_name_walk = (_tag: TagStructElementData): void => { }
    tag_name_walk = (tag: TagStructElementData): void => {
        const tag_children = tag.children
        if (tag.id === id) {
            target_struct_object = tag
        } else if (tag_children) {
            tag_children.forEach(child_tag => {
                if (child_tag) {
                    tag_name_walk(child_tag)
                }
            })
        }
    }
    tag_name_walk(cloned_application_config.value.tag_struct)

    if (!target_struct_object) {
        return
    }
    confirm_delete_tag_struct_dialog.value?.show(target_struct_object)
}
function delete_tag_struct(id: string): void {
    let tag_name_walk = (_tag: TagStructElementData): boolean => false
    tag_name_walk = (tag: TagStructElementData): boolean => {
        const tag_children = tag.children
        if (tag.id === id) {
            return true
        } else if (tag_children) {
            for (let i = 0; i < tag_children.length; i++) {
                const child_tag = tag_children[i]
                if (child_tag.children) {
                    if (tag_name_walk(child_tag)) {
                        tag_children.splice(i, 1)
                        return false
                    }
                }
            }
        }
        return false
    }
    tag_name_walk(cloned_application_config.value.tag_struct)
}
</script>
<style lang="css" scoped>
.tag_struct_root {
    max-height: 80vh;
    overflow-y: scroll;
}
</style>