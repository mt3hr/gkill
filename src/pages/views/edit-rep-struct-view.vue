<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t('EDIT_REP_STRUCT_TITLE') }}
        </v-card-title>
        <div class="rep_struct_root">
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api"
                :folder_name="i18n.global.t('REP_TITLE')" :is_open="true"
                :struct_obj="cloned_application_config.rep_struct" :is_editable="true" :is_root="true"
                :is_show_checkbox="false"
                @dblclicked_item="(e: MouseEvent, id: string | null) => { if (id) show_edit_rep_struct_dialog(id) }"
                @contextmenu_item="show_rep_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_rep_struct_element_dialog">{{
                        i18n.global.t("ADD_REP_TITLE") }}</v-btn>
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
        <AddNewRepStructElementDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :is_open="true"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_add_rep_struct_element="add_rep_struct_element" ref="add_new_rep_struct_element_dialog" />
        <EditRepStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_update_rep_struct="update_rep_struct" ref="edit_rep_struct_element_dialog" />
        <RepStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="rep_struct_context_menu"
            @requested_edit_rep="(...id: any[]) => show_edit_rep_struct_dialog(id[0] as string)"
            @requested_delete_rep="(...id: any[]) => show_confirm_delete_rep_struct_dialog(id[0] as string)" />
        <ConfirmDeleteRepStructDialog ref="confirm_delete_rep_struct_dialog" :application_config="application_config"
            :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_delete_rep="(...id: any[]) => delete_rep_struct(id[0] as string)" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditRepStructViewEmits } from './edit-rep-struct-view-emits'
import type { EditRepStructViewProps } from './edit-rep-struct-view-props'
import AddNewRepStructElementDialog from '../dialogs/add-new-rep-struct-element-dialog.vue'
import EditRepStructElementDialog from '../dialogs/edit-rep-struct-element-dialog.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import FoldableStruct from './foldable-struct.vue'
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'

import RepStructContextMenu from './rep-struct-context-menu.vue'
import ConfirmDeleteRepStructDialog from '../dialogs/confirm-delete-rep-struct-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

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
async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
    await cloned_application_config.value.append_not_found_reps()
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
    let target_struct_object: RepStructElementData | null = null
    let rep_name_walk = (_rep: RepStructElementData): void => { }
    rep_name_walk = (rep: RepStructElementData): void => {
        const rep_children = rep.children
        if (rep.id === id) {
            target_struct_object = rep
        } else if (rep_children) {
            rep_children.forEach(child_rep => {
                if (child_rep) {
                    rep_name_walk(child_rep)
                }
            })
        }
    }
    rep_name_walk(cloned_application_config.value.rep_struct)

    if (!target_struct_object) {
        return
    }

    edit_rep_struct_element_dialog.value?.show(target_struct_object)
}

function update_rep_struct(rep_struct_obj: RepStructElementData): void {
    let rep_name_walk = (_rep: RepStructElementData): boolean => false
    rep_name_walk = (rep: RepStructElementData): boolean => {
        const rep_children = rep.children
        if (rep.id === rep_struct_obj.id) {
            return true
        } else if (rep_children) {
            for (let i = 0; i < rep_children.length; i++) {
                const child_rep = rep_children[i]
                if (child_rep.children) {
                    if (rep_name_walk(child_rep)) {
                        rep_children[i] = rep_struct_obj
                        return false
                    }
                }
            }
        }
        return false
    }
    rep_name_walk(cloned_application_config.value.rep_struct)
}

async function apply(): Promise<void> {
    emits('requested_apply_rep_struct', cloned_application_config.value.rep_struct)
    nextTick(() => emits('requested_close_dialog'))
}
function show_add_new_rep_struct_element_dialog(): void {
    add_new_rep_struct_element_dialog.value?.show()
}
async function add_rep_struct_element(rep_struct_element: RepStructElementData): Promise<void> {
    cloned_application_config.value.rep_struct.children?.push(rep_struct_element)
}
function show_confirm_delete_rep_struct_dialog(id: string): void {
    let target_struct_object: RepStructElementData | null = null
    let rep_name_walk = (_rep: RepStructElementData): void => { }
    rep_name_walk = (rep: RepStructElementData): void => {
        const rep_children = rep.children
        if (rep.id === id) {
            target_struct_object = rep
        } else if (rep_children) {
            rep_children.forEach(child_rep => {
                if (child_rep) {
                    rep_name_walk(child_rep)
                }
            })
        }
    }
    rep_name_walk(cloned_application_config.value.rep_struct)

    if (!target_struct_object) {
        return
    }
    confirm_delete_rep_struct_dialog.value?.show(target_struct_object)

}
function delete_rep_struct(id: string): void {
    let rep_name_walk = (_rep: RepStructElementData): boolean => false
    rep_name_walk = (rep: RepStructElementData): boolean => {
        const rep_children = rep.children
        if (rep.id === id) {
            return true
        } else if (rep_children) {
            for (let i = 0; i < rep_children.length; i++) {
                const child_rep = rep_children[i]
                if (child_rep.children) {
                    if (rep_name_walk(child_rep)) {
                        rep_children.splice(i, 1)
                        return false
                    }
                }
            }
        }
        return false
    }
    rep_name_walk(cloned_application_config.value.rep_struct)
}
</script>
<style lang="css" scoped>
.rep_struct_root {
    max-height: 80vh;
    overflow-y: scroll;
}
</style>