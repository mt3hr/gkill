<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ $t("ADD_KFTL_TEMPLATE_STRUCT_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="title" :label="$t('TEMPLATE_NAME_TITLE')" />
        <v-textarea v-model="template" :label="$t('TEMPLATE_CONTENT_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="emits_kftl_template_name">{{ $t("ADD_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">
                        {{ $t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { KFTLTemplateStructElementData } from '@/classes/datas/config/kftl-template-struct-element-data';
import { type Ref, ref } from 'vue';
import type { AddNewKFTLTemplateStructElementViewEmits } from './add-new-kftl-template-struct-element-view-emits'
import type { AddNewKFTLTemplateStructElementViewProps } from './add-new-kftl-template-struct-element-view-props'
import { GkillError } from '@/classes/api/gkill-error';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';
import { useI18n } from 'vue-i18n'

import { i18n } from '@/i18n'

const props = defineProps<AddNewKFTLTemplateStructElementViewProps>()
const emits = defineEmits<AddNewKFTLTemplateStructElementViewEmits>()

defineExpose({ reset_kftl_template_name })

const title: Ref<string> = ref("")
const template: Ref<string> = ref("")

function emits_kftl_template_name(): void {
    if (title.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.kftl_template_title_is_blank
        error.error_message = i18n.global.t("KFTL_TEMPLATE_NAME_IS_BLANK_MESSAGE")
        emits('received_errors', [error])
        return
    }

    if (template.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.kftl_template_content_is_blank
        error.error_message = i18n.global.t("KFTL_TEMPLATE_CONTENT_IS_BLANK_MESSAGE")
        emits('received_errors', [error])
        return
    }

    const kftl_template_struct_element = new KFTLTemplateStructElementData()
    kftl_template_struct_element.id = props.gkill_api.generate_uuid()
    kftl_template_struct_element.key = title.value
    kftl_template_struct_element.title = title.value
    kftl_template_struct_element.template = template.value
    emits('requested_add_kftl_template_struct_element', kftl_template_struct_element)
    emits('requested_close_dialog')
}

function reset_kftl_template_name(): void {
    title.value = ""
    template.value = ""
}
</script>
