<template>
    <v-card class="pa-2">
        <v-card-title>
            データタイプ追加
        </v-card-title>
        <v-text-field class="input" type="text" v-model="rep_type_name" label="データタイプ名" />
        <v-checkbox v-model="check_when_inited" hide-detail label="初期化時チェック" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn color="primary" @click="emits_rep_type_name">追加</v-btn>
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
import { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data';
import { type Ref, ref } from 'vue';
import type { AddNewRepTypeStructElementViewEmits } from './add-new-rep-type-struct-element-view-emits'
import type { AddNewRepTypeStructElementViewProps } from './add-new-rep-type-struct-element-view-props'
import { GkillError } from '@/classes/api/gkill-error';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';

const props = defineProps<AddNewRepTypeStructElementViewProps>()
const emits = defineEmits<AddNewRepTypeStructElementViewEmits>()

defineExpose({ reset_rep_type_name })

const rep_type_name: Ref<string> = ref("")
const check_when_inited: Ref<boolean> = ref(true)

function emits_rep_type_name(): void {
    if (rep_type_name.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.rep_type_struct_title_is_blank
        error.error_message = "データタイプ名が入力されていません"
        emits('received_errors', [error])
        return
    }

    const rep_type_struct_element = new RepTypeStructElementData()
    rep_type_struct_element.id = props.gkill_api.generate_uuid()
    rep_type_struct_element.check_when_inited = check_when_inited.value
    rep_type_struct_element.children = null
    rep_type_struct_element.indeterminate = false
    rep_type_struct_element.key = rep_type_name.value
    rep_type_struct_element.rep_type_name = rep_type_name.value
    emits('requested_add_rep_type_struct_element', rep_type_struct_element)
    emits('requested_close_dialog')
}

function reset_rep_type_name(): void {
    rep_type_name.value = ""
    check_when_inited.value = true
}
</script>
