<template>
    <v-card class="pa-2">
        <v-card-title>
            タグ追加
        </v-card-title>
        <v-text-field class="input" type="text" v-model="tag_name" label="タグ名" />
        <v-checkbox v-model="check_when_inited" hide-detail label="初期化時チェック" />
        <v-checkbox v-model="is_force_hide" hide-detail label="非表示優先" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn color="primary" @click="emits_tag_name">追加</v-btn>
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
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data';
import { type Ref, ref } from 'vue';
import type { AddNewTagStructElementViewEmits } from './add-new-tag-struct-element-view-emits'
import type { AddNewTagStructElementViewProps } from './add-new-tag-struct-element-view-props'

const props = defineProps<AddNewTagStructElementViewProps>()
const emits = defineEmits<AddNewTagStructElementViewEmits>()

defineExpose({ reset_tag_name })

const tag_name: Ref<string> = ref("")
const check_when_inited: Ref<boolean> = ref(true)
const is_force_hide: Ref<boolean> = ref(false)

function emits_tag_name(): void {
    const tag_struct_element = new TagStructElementData()
    tag_struct_element.id = props.gkill_api.generate_uuid()
    tag_struct_element.check_when_inited = check_when_inited.value
    tag_struct_element.is_force_hide = is_force_hide.value
    tag_struct_element.children = null
    tag_struct_element.indeterminate = false
    tag_struct_element.key = tag_name.value
    tag_struct_element.tag_name = tag_name.value
    emits('requested_add_tag_struct_element', tag_struct_element)
    emits('requested_close_dialog')
}

function reset_tag_name(): void {
    tag_name.value = ""
    check_when_inited.value = true
    is_force_hide.value = false
}
</script>
