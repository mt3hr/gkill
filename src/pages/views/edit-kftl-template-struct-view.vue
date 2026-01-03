<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("EDIT_KFTL_TEMPLATE_STRUCT_TITLE") }}
        </v-card-title>
        <div class="kftl_template_struct_root">
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api"
                :folder_name="i18n.global.t('KFTL_TEMPLATE_STRUCT_ELEMENT_TITLE')" :is_open="true"
                :struct_obj="application_config.kftl_template_struct" :is_editable="true" :is_root="true"
                :is_show_checkbox="false"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @dblclicked_item="(e: MouseEvent, id: string | null) => { if (id) show_edit_kftl_template_struct_dialog(id) }"
                @contextmenu_item="show_kftl_template_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_kftl_template_struct_element_dialog">{{
                        i18n.global.t("ADD_KFTL_TEMPLATE_STRUCT_ELEMENT_TITLE") }}</v-btn>
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
        <AddNewKFTLTemplateStructElementDialog :application_config="application_config" :folder_name="''"
            :gkill_api="gkill_api" :is_open="true"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_add_kftl_template_struct_element="add_kftl_template_struct_element"
            ref="add_new_kftl_template_struct_element_dialog" />
        <EditKFTLTemplateStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_update_kftl_template_struct="update_kftl_template_struct"
            ref="edit_kftl_template_struct_element_dialog" />
        <KFTLTemplateStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="kftl_template_struct_context_menu"
            @requested_edit_kftl_template="(...id: any[]) => show_edit_kftl_template_struct_dialog(id[0] as string)"
            @requested_delete_kftl_template="(...id: any[]) => show_confirm_delete_kftl_template_struct_dialog(id[0] as string)" />
        <ConfirmDeleteKFTLTemplateStructDialog ref="confirm_delete_kftl_template_struct_dialog"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_delete_kftl_template="(...id: any[]) => delete_kftl_template_struct(id[0] as string)" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditKFTLTemplateStructViewEmits } from './edit-kftl-template-struct-view-emits.ts'
import type { EditKFTLTemplateStructViewProps } from './edit-kftl-template-struct-view-props.ts'
import AddNewKFTLTemplateStructElementDialog from '../dialogs/add-new-kftl-template-struct-element-dialog.vue'
import EditKFTLTemplateStructElementDialog from '../dialogs/edit-kftl-template-struct-element-dialog.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import FoldableStruct from './foldable-struct.vue'
import type { FoldableStructModel } from './foldable-struct-model'
import AddNewFoloderDialog from '../dialogs/add-new-foloder-dialog.vue'
import { KFTLTemplateStructElementData } from '@/classes/datas/config/kftl-template-struct-element-data'

import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import KFTLTemplateStructContextMenu from './kftl-template-struct-context-menu.vue'
import ConfirmDeleteKFTLTemplateStructDialog from '../dialogs/confirm-delete-kftl-template-struct-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error.js'
import type { GkillMessage } from '@/classes/api/gkill-message.js'

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
async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
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
    let target_struct_object: KFTLTemplateStructElementData | null = null
    let kftl_template_walk = (_kftl_template: KFTLTemplateStructElementData): void => { }
    kftl_template_walk = (kftl_template: KFTLTemplateStructElementData): void => {
        const kftl_template_children = kftl_template.children
        if (kftl_template.id === id) {
            target_struct_object = kftl_template
        } else if (kftl_template_children) {
            kftl_template_children.forEach(child_kftl_template => {
                if (child_kftl_template) {
                    kftl_template_walk(child_kftl_template)
                }
            })
        }
    }
    kftl_template_walk(cloned_application_config.value.kftl_template_struct)

    if (!target_struct_object) {
        return
    }

    edit_kftl_template_struct_element_dialog.value?.show(target_struct_object)
}

function update_kftl_template_struct(kftl_template_struct_obj: KFTLTemplateStructElementData): void {
    let kftl_template_walk = (_kftl_template: KFTLTemplateStructElementData): boolean => false
    kftl_template_walk = (kftl_template: KFTLTemplateStructElementData): boolean => {
        const kftl_template_children = kftl_template.children
        if (kftl_template.id === kftl_template.id) {
            return true
        } else if (kftl_template_children) {
            for (let i = 0; i < kftl_template_children.length; i++) {
                const child_kftl_template = kftl_template_children[i]
                if (child_kftl_template.children) {
                    if (kftl_template_walk(child_kftl_template)) {
                        kftl_template_children[i] = kftl_template_struct_obj
                        return false
                    }
                }
            }
        }
        return false
    }
    kftl_template_walk(cloned_application_config.value.kftl_template_struct)
}

async function apply(): Promise<void> {
    emits('requested_apply_kftl_template_struct', cloned_application_config.value.kftl_template_struct)
    nextTick(() => emits('requested_close_dialog'))
}
function show_add_new_kftl_template_struct_element_dialog(): void {
    add_new_kftl_template_struct_element_dialog.value?.show()
}
function show_add_new_folder_dialog(): void {
    add_new_folder_dialog.value?.show()
}
async function add_folder_struct_element(folder_struct_element: FolderStructElementData): Promise<void> {
    const kftl_template_struct_element = new KFTLTemplateStructElementData()
    kftl_template_struct_element.id = folder_struct_element.id
    kftl_template_struct_element.title = folder_struct_element.folder_name
    kftl_template_struct_element.children = new Array<KFTLTemplateStructElementData>()
    kftl_template_struct_element.key = folder_struct_element.folder_name
    kftl_template_struct_element.name = folder_struct_element.folder_name
    cloned_application_config.value.kftl_template_struct.children?.push(kftl_template_struct_element)
}
async function add_kftl_template_struct_element(kftl_template_struct_element: KFTLTemplateStructElementData): Promise<void> {
    cloned_application_config.value.kftl_template_struct.children?.push(kftl_template_struct_element)
}
function show_confirm_delete_kftl_template_struct_dialog(id: string): void {
    let target_struct_object: KFTLTemplateStructElementData | null = null
    let kftl_template_walk = (_kftl_template_struct: KFTLTemplateStructElementData): void => { }
    kftl_template_walk = (kftl_template_struct: KFTLTemplateStructElementData): void => {
        const kftl_template_children = kftl_template_struct.children
        if (kftl_template_struct.id === id) {
            target_struct_object = kftl_template_struct
        } else if (kftl_template_children) {
            kftl_template_children.forEach(child_kftl_template => {
                if (child_kftl_template) {
                    kftl_template_walk(child_kftl_template)
                }
            })
        }
    }
    kftl_template_walk(cloned_application_config.value.kftl_template_struct)

    if (!target_struct_object) {
        return
    }
    confirm_delete_kftl_template_struct_dialog.value?.show(target_struct_object)
}
function delete_kftl_template_struct(id: string): void {
    let kftl_template_walk = (_kftl_template_struct: KFTLTemplateStructElementData): boolean => false
    kftl_template_walk = (kftl_template_struct: KFTLTemplateStructElementData): boolean => {
        const kftl_template_children = kftl_template_struct.children
        if (kftl_template_struct.id === id) {
            return true
        } else if (kftl_template_children) {
            for (let i = 0; i < kftl_template_children.length; i++) {
                const child_kftl_template = kftl_template_children[i]
                if (child_kftl_template.children) {
                    if (kftl_template_walk(child_kftl_template)) {
                        kftl_template_children.splice(i, 1)
                        return false
                    }
                }
            }
        }
        return false
    }
    kftl_template_walk(cloned_application_config.value.kftl_template_struct)
}
</script>
<style lang="css" scoped>
.kftl_template_struct_root {
    max-height: 80vh;
    overflow-y: scroll;
}
</style>