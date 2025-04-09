<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ $t("ADD_REP_TYPE_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="rep_type_name" :label="$t('REP_TYPE_TITLE')" />
        <v-checkbox v-model="check_when_inited" hide-detail :label="$t('CHECK_WHEN_INITED_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="emits_rep_type_name">{{ $t("ADD_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ $t("CANCEL_TITLE")
                    }}</v-btn>
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
import { useI18n } from 'vue-i18n'

import { i18n } from '@/i18n'

const props = defineProps<AddNewRepTypeStructElementViewProps>()
const emits = defineEmits<AddNewRepTypeStructElementViewEmits>()

defineExpose({ reset_rep_type_name })

const rep_type_name: Ref<string> = ref("")
const check_when_inited: Ref<boolean> = ref(true)

function emits_rep_type_name(): void {
    if (rep_type_name.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.rep_type_struct_title_is_blank
        error.error_message = i18n.global.t("REP_TYPE_IS_BLANK_MESSAGE")
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
