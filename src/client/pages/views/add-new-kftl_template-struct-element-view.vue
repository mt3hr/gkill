<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("ADD_KFTL_TEMPLATE_STRUCT_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="title" :label="i18n.global.t('TEMPLATE_NAME_TITLE')" />
        <v-textarea v-model="template" :label="i18n.global.t('TEMPLATE_CONTENT_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="emits_kftl_template_name">{{ i18n.global.t("ADD_TITLE")
                        }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">
                        {{ i18n.global.t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AddNewKFTLTemplateStructElementViewEmits } from './add-new-kftl-template-struct-element-view-emits'
import type { AddNewKFTLTemplateStructElementViewProps } from './add-new-kftl-template-struct-element-view-props'
import { useAddNewKftlTemplateStructElementView } from '@/classes/use-add-new-kftl-template-struct-element-view'

const props = defineProps<AddNewKFTLTemplateStructElementViewProps>()
const emits = defineEmits<AddNewKFTLTemplateStructElementViewEmits>()

const {
    title,
    template,
    emits_kftl_template_name,
    reset_kftl_template_name,
} = useAddNewKftlTemplateStructElementView({ props, emits })

defineExpose({ reset_kftl_template_name })
</script>
