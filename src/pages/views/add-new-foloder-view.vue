<template>
    <v-card class="pa-2">
        <v-card-title>
            フォルダ追加
        </v-card-title>
        <v-text-field class="input" type="text" v-model="folder_name" label="フォルダ名" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn color="primary" @click="emits_folder">追加</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn color="primary" @click="emits('requested_close_dialog')">キャンセル</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { ref, type Ref } from 'vue';
import type { AddNewFoloderViewEmits } from './add-new-foloder-view-emits'
import type { AddNewFoloderViewProps } from './add-new-foloder-view-props'
import { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data';
import { GkillAPI } from '@/classes/api/gkill-api';

const props = defineProps<AddNewFoloderViewProps>()
const emits = defineEmits<AddNewFoloderViewEmits>()

defineExpose({ reset_folder_name })

const folder_name: Ref<string> = ref("")

function emits_folder(): void {
    const folder_struct_element = new FolderStructElementData()
    folder_struct_element.id = props.gkill_api.generate_uuid()
    folder_struct_element.folder_name = folder_name.value
    emits('requested_add_new_folder', folder_struct_element)
    emits('requested_close_dialog')
}

function reset_folder_name(): void {
    folder_name.value = ""
}
</script>
