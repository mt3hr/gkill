<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t('EDIT_TAG_STRUCT_ELEMENT_TITLE') }}
        </v-card-title>
        <p>{{ struct_obj.tag_name }}</p>
        <v-checkbox v-model="check_when_inited" hide-detail :label="i18n.global.t('CHECK_WHEN_INITED_TITLE')" />
        <v-checkbox v-model="is_force_hide" hide-detail :label="i18n.global.t('IS_FORCE_HIDE_TITLE')" />
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
import type { EditTagStructElementViewEmits } from './edit-tag-struct-element-view-emits'
import type { EditTagStructElementViewProps } from './edit-tag-struct-element-view-props'
import { useEditTagStructElementView } from '@/classes/use-edit-tag-struct-element-view'

const props = defineProps<EditTagStructElementViewProps>()
const emits = defineEmits<EditTagStructElementViewEmits>()

const {
    check_when_inited,
    is_force_hide,
    apply,
} = useEditTagStructElementView({ props, emits })
</script>
