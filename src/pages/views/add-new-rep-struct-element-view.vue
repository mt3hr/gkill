<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ $t("ADD_REP_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="rep_name" :label="$t('REP_TITLE')" />
        <v-checkbox v-model="check_when_inited" hide-detail :label="$t('IS_FORCE_HIDE_TITLE')" />
        <v-checkbox v-model="ignore_check_rep_rykv" hide-detail :label="$t('IGNORE_CHECK_REP_RYKV_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="emits_rep_name">{{ $t("ADD_TITLE") }}</v-btn>
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
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data';
import { type Ref, ref } from 'vue';
import type { AddNewRepStructElementViewEmits } from './add-new-rep-struct-element-view-emits'
import type { AddNewRepStructElementViewProps } from './add-new-rep-struct-element-view-props'
import { GkillError } from '@/classes/api/gkill-error';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';
import { useI18n } from 'vue-i18n'

import { i18n } from '@/i18n'

const props = defineProps<AddNewRepStructElementViewProps>()
const emits = defineEmits<AddNewRepStructElementViewEmits>()

defineExpose({ reset_rep_name })

const rep_name: Ref<string> = ref("")
const check_when_inited: Ref<boolean> = ref(true)
const ignore_check_rep_rykv: Ref<boolean> = ref(false)

function emits_rep_name(): void {
    if (rep_name.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.rep_struct_title_is_blank
        error.error_message = i18n.global.t("REP_IS_BLANK_MESSAGE")
        emits('received_errors', [error])
        return
    }

    const rep_struct_element = new RepStructElementData()
    rep_struct_element.id = props.gkill_api.generate_uuid()
    rep_struct_element.check_when_inited = check_when_inited.value
    rep_struct_element.ignore_check_rep_rykv = ignore_check_rep_rykv.value
    rep_struct_element.children = null
    rep_struct_element.indeterminate = false
    rep_struct_element.key = rep_name.value
    rep_struct_element.rep_name = rep_name.value
    emits('requested_add_rep_struct_element', rep_struct_element)
    emits('requested_close_dialog')
}

function reset_rep_name(): void {
    rep_name.value = ""
    check_when_inited.value = true
    ignore_check_rep_rykv.value = false
}
</script>
