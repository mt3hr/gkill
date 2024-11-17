<template>
    <v-card>
        <v-card-title>
            タグ構造
        </v-card-title>
        <FoldableStruct :application_config="application_config" :gkill_api="gkill_api" :folder_name="'タグ'"
            :is_open="true" :struct_obj="cloned_application_config.parsed_tag_struct" :is_editable="true"
            :is_root="true" :is_show_checkbox="false" @dblclicked_item="(id: string) => show_edit_tag_struct_dialog(id)"
            ref="foldable_struct" />
        <v-card-action>
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
        <AddNewTagStructElementDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :is_open="true" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_add_tag_struct_element="'//TODO'" />
        <EditTagStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_tag_struct="update_tag_struct" ref="edit_tag_struct_element_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref, watch } from 'vue'
import type { EditTagStructViewEmits } from './edit-tag-struct-view-emits'
import type { EditTagStructViewProps } from './edit-tag-struct-view-props'
import AddNewTagStructElementDialog from '../dialogs/add-new-tag-struct-element-dialog.vue'
import EditTagStructElementDialog from '../dialogs/edit-tag-struct-element-dialog.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import FoldableStruct from './foldable-struct.vue'
import type { TagStruct } from '@/classes/datas/config/tag-struct'
import type { FoldableStructModel } from './foldable-struct-model'
import { UpdateTagStructRequest } from '@/classes/api/req_res/update-tag-struct-request'
import { GkillAPI } from '@/classes/api/gkill-api'

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null);
const edit_tag_struct_element_dialog = ref<InstanceType<typeof EditTagStructElementDialog> | null>(null);

const props = defineProps<EditTagStructViewProps>()
const emits = defineEmits<EditTagStructViewEmits>()
defineExpose({ reload_cloned_application_config })

watch(() => props.application_config, () => reload_cloned_application_config())

const cloned_application_config: Ref<ApplicationConfig> = ref(await props.application_config.clone())

cloned_application_config.value.parse_tag_struct()

async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = await props.application_config.clone()
    cloned_application_config.value.parse_tag_struct()
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
    cloned_application_config.value.parse_tag_struct()
    if (cloned_application_config.value.parsed_tag_struct.children) {
        update_seq(cloned_application_config.value.parsed_tag_struct.children)
    }
}

function update_seq(tag_struct: Array<FoldableStructModel>): void {
    // 並び順再決定
    let f = (struct: FoldableStructModel, seq: number) => { }
    let func = (struct: FoldableStructModel, seq: number) => {
        for (let i = 0; i < cloned_application_config.value.tag_struct.length; i++) {
            if (struct.id === cloned_application_config.value.tag_struct[i].id) {
                cloned_application_config.value.tag_struct[i].seq = seq
            }
        }
        if (struct.children) {
            for (let i = 0; i < struct.children.length; i++) {
                f(struct.children[i], i)
            }
        }
    }
    f = func
    for (let i = 0; i < tag_struct.length; i++) {
        f(tag_struct[i], i)
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
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.tag_struct = cloned_application_config.value.tag_struct
    const res = await GkillAPI.get_instance().update_tag_struct(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('requested_close_dialog')
}
</script>
