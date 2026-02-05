<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("EDIT_REP_TYPE_STRUCT_TITLE") }}
        </v-card-title>
        <div class="rep_type_struct_root">
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api"
                :folder_name="i18n.global.t('REP_TYPE_TITLE')" :is_open="true"
                :struct_obj="cloned_application_config.rep_type_struct" :is_editable="true" :is_root="true"
                :is_show_checkbox="false"
                @dblclicked_item="(e: MouseEvent, id: string | null) => { if (id) show_edit_rep_type_struct_dialog(id) }"
                @contextmenu_item="show_rep_type_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_rep_type_struct_element_dialog">{{
                        i18n.global.t("ADD_REP_TYPE_TITLE") }}</v-btn>
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
                        i18n.global.t("CANCEL_TITLE")
                        }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AddNewFoloderDialog :application_config="application_config" :gkill_api="gkill_api"
            @requested_add_new_folder="add_folder_struct_element"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="add_new_folder_dialog" />
        <AddNewRepTypeStructElementDialog :application_config="application_config" :folder_name="''"
            :gkill_api="gkill_api" :is_open="true"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_add_rep_type_struct_element="add_rep_type_struct_element"
            ref="add_new_rep_type_struct_element_dialog" />
        <EditRepTypeStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_update_rep_type_struct="update_rep_type_struct" ref="edit_rep_type_struct_element_dialog" />
        <RepTypeStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="rep_type_struct_context_menu"
            @requested_edit_rep_type="(...id: any[]) => show_edit_rep_type_struct_dialog(id[0] as string)"
            @requested_delete_rep_type="(...id: any[]) => show_confirm_delete_rep_type_struct_dialog(id[0] as string)" />
        <ConfirmDeleteRepTypeStructDialog ref="confirm_delete_rep_type_struct_dialog"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_delete_rep_type="(...id: any[]) => delete_rep_type_struct(id[0] as string)" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditRepTypeStructViewEmits } from './edit-rep-type-struct-view-emits.js'
import type { EditRepTypeStructViewProps } from './edit-rep-type-struct-view-props.js'
import AddNewRepTypeStructElementDialog from '../dialogs/add-new-rep-type-struct-element-dialog.vue'
import EditRepTypeStructElementDialog from '../dialogs/edit-rep-type-struct-element-dialog.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import FoldableStruct from './foldable-struct.vue'
import AddNewFoloderDialog from '../dialogs/add-new-foloder-dialog.vue'
import { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'

import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import RepTypeStructContextMenu from './rep-type-struct-context-menu.vue'
import ConfirmDeleteRepTypeStructDialog from '../dialogs/confirm-delete-rep-type-struct-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error.js'
import type { GkillMessage } from '@/classes/api/gkill-message.js'

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null);
const edit_rep_type_struct_element_dialog = ref<InstanceType<typeof EditRepTypeStructElementDialog> | null>(null);
const add_new_folder_dialog = ref<InstanceType<typeof AddNewFoloderDialog> | null>(null);
const add_new_rep_type_struct_element_dialog = ref<InstanceType<typeof AddNewRepTypeStructElementDialog> | null>(null);
const rep_type_struct_context_menu = ref<InstanceType<typeof RepTypeStructContextMenu> | null>(null);
const confirm_delete_rep_type_struct_dialog = ref<InstanceType<typeof ConfirmDeleteRepTypeStructDialog> | null>(null);

const props = defineProps<EditRepTypeStructViewProps>()
const emits = defineEmits<EditRepTypeStructViewEmits>()
defineExpose({ reload_cloned_application_config })

watch(() => props.application_config, () => reload_cloned_application_config())
const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())
async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
    await cloned_application_config.value.append_not_found_rep_types()
}

function show_rep_type_contextmenu(e: MouseEvent, id: string | null): void {
    if (id) {
        rep_type_struct_context_menu.value?.show(e, id)
    }
}

function show_edit_rep_type_struct_dialog(id: string): void {
    if (!foldable_struct.value) {
        return
    }
    let target_struct_object: RepTypeStructElementData | null = null
    let rep_type_name_walk = (_rep_type: RepTypeStructElementData): void => { }
    rep_type_name_walk = (rep_type: RepTypeStructElementData): void => {
        const rep_type_children = rep_type.children
        if (rep_type.id === id) {
            target_struct_object = rep_type
        } else if (rep_type_children) {
            rep_type_children.forEach(child_rep_type => {
                if (child_rep_type) {
                    rep_type_name_walk(child_rep_type)
                }
            })
        }
    }
    rep_type_name_walk(cloned_application_config.value.rep_type_struct)

    if (!target_struct_object) {
        return
    }

    edit_rep_type_struct_element_dialog.value?.show(target_struct_object)
}

function update_rep_type_struct(rep_type_struct_obj: RepTypeStructElementData): void {
    let rep_type_name_walk = (_rep_type: RepTypeStructElementData): boolean => false
    rep_type_name_walk = (rep_type: RepTypeStructElementData): boolean => {
        const rep_type_children = rep_type.children
        if (rep_type.id === rep_type_struct_obj.id) {
            return true
        } else if (rep_type_children) {
            for (let i = 0; i < rep_type_children.length; i++) {
                const child_rep_type = rep_type_children[i]
                if (rep_type_name_walk(child_rep_type)) {
                    rep_type_children.splice(i, 1, rep_type_struct_obj)
                    return false
                }
            }
        }
        return false
    }
    rep_type_name_walk(cloned_application_config.value.rep_type_struct)
}

async function apply(): Promise<void> {
    emits('requested_apply_rep_type_struct', cloned_application_config.value.rep_type_struct)
    nextTick(() => emits('requested_close_dialog'))
}

function show_add_new_rep_type_struct_element_dialog(): void {
    add_new_rep_type_struct_element_dialog.value?.show()
}

function show_add_new_folder_dialog(): void {
    add_new_folder_dialog.value?.show()
}
async function add_folder_struct_element(folder_struct_element: FolderStructElementData): Promise<void> {
    const rep_type_struct_element = new RepTypeStructElementData()
    rep_type_struct_element.id = folder_struct_element.id
    rep_type_struct_element.is_dir = true
    rep_type_struct_element.check_when_inited = false
    rep_type_struct_element.rep_type_name = folder_struct_element.folder_name
    rep_type_struct_element.children = new Array<RepTypeStructElementData>()
    rep_type_struct_element.key = folder_struct_element.folder_name
    cloned_application_config.value.rep_type_struct.children?.push(rep_type_struct_element)
}
async function add_rep_type_struct_element(rep_type_struct_element: RepTypeStructElementData): Promise<void> {
    cloned_application_config.value.rep_type_struct.children?.push(rep_type_struct_element)
}
function show_confirm_delete_rep_type_struct_dialog(id: string): void {
    let target_struct_object: RepTypeStructElementData | null = null
    let rep_type_name_walk = (_rep_type: RepTypeStructElementData): void => { }
    rep_type_name_walk = (rep_type: RepTypeStructElementData): void => {
        const rep_type_children = rep_type.children
        if (rep_type.id === id) {
            target_struct_object = rep_type
        } else if (rep_type_children) {
            rep_type_children.forEach(child_rep_type => {
                if (child_rep_type) {
                    rep_type_name_walk(child_rep_type)
                }
            })
        }
    }
    rep_type_name_walk(cloned_application_config.value.rep_type_struct)

    if (!target_struct_object) {
        return
    }
    confirm_delete_rep_type_struct_dialog.value?.show(target_struct_object)
}
function delete_rep_type_struct(id: string): void {
    let rep_type_name_walk = (_rep_type: RepTypeStructElementData): boolean => false
    rep_type_name_walk = (rep_type: RepTypeStructElementData): boolean => {
        const rep_type_children = rep_type.children
        if (rep_type.id === id) {
            return true
        } else if (rep_type_children) {
            for (let i = 0; i < rep_type_children.length; i++) {
                const child_rep_type = rep_type_children[i]
                if (rep_type_name_walk(child_rep_type)) {
                    rep_type_children.splice(i, 1)
                    return false
                }
            }
        }
        return false
    }
    rep_type_name_walk(cloned_application_config.value.rep_type_struct)
}
</script>
<style lang="css" scoped>
.rep_type_struct_root {
    max-height: 80vh;
    overflow-y: scroll;
}
</style>