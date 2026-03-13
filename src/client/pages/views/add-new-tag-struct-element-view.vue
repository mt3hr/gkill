<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("ADD_TAG_STRUCT_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="tag_name" :label="i18n.global.t('TAG_TITLE')" />
        <v-checkbox v-model="check_when_inited" hide-detail :label="i18n.global.t('CHECK_WHEN_INITED_TITLE')" />
        <v-checkbox v-model="is_force_hide" hide-detail :label="i18n.global.t('IS_FORCE_HIDE_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="emits_tag_name">{{ i18n.global.t('ADD_TITLE') }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{
                        i18n.global.t('CANCEL_TITLE') }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AddNewTagStructElementViewEmits } from './add-new-tag-struct-element-view-emits'
import type { AddNewTagStructElementViewProps } from './add-new-tag-struct-element-view-props'
import { useAddNewTagStructElementView } from '@/classes/use-add-new-tag-struct-element-view'

const props = defineProps<AddNewTagStructElementViewProps>()
const emits = defineEmits<AddNewTagStructElementViewEmits>()

const {
    tag_name,
    check_when_inited,
    is_force_hide,
    emits_tag_name,
    reset_tag_name,
} = useAddNewTagStructElementView({ props, emits })

defineExpose({ reset_tag_name })
</script>
