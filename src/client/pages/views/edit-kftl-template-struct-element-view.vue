<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("EDIT_KFTL_TEMPLATE_STRUCT_ELEMENT_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="title" :label="i18n.global.t('TEMPLATE_NAME_TITLE')" />
        <v-textarea v-model="template" :label="i18n.global.t('TEMPLATE_CONTENT_TITLE')" />
        <v-card-action>
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
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { EditKFTLTemplateStructElementViewEmits } from './edit-kftl-template-struct-element-view-emits'
import type { EditKFTLTemplateStructElementViewProps } from './edit-kftl-template-struct-element-view-props'
import { useEditKFTLTemplateStructElementView } from '@/classes/use-edit-kftl-template-struct-element-view'

const props = defineProps<EditKFTLTemplateStructElementViewProps>()
const emits = defineEmits<EditKFTLTemplateStructElementViewEmits>()

const {
    title,
    template,
    apply,
} = useEditKFTLTemplateStructElementView({ props, emits })
</script>
