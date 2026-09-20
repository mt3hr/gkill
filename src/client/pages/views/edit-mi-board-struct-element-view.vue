<template>
    <v-card variant="flat">
        <v-card-title>
            {{ i18n.global.t("EDIT_MI_BOARD_STRUCT_ELEMENT_TITLE") }}
        </v-card-title>
        <p>{{ struct_obj.board_name }}</p>
        <v-textarea v-model="description" :label="i18n.global.t('DESCRIPTION_TITLE')"
            :hint="i18n.global.t('STRUCT_DESCRIPTION_HINT')" persistent-hint auto-grow rows="2" />
        <v-card-action>
            <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
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
import type { EditMiBoardStructElementViewEmits } from './edit-mi-board-struct-element-view-emits'
import type { EditMiBoardStructElementViewProps } from './edit-mi-board-struct-element-view-props'
import { useEditMiBoardStructElementView } from '@/classes/use-edit-mi-board-struct-element-view'

const props = defineProps<EditMiBoardStructElementViewProps>()
const emits = defineEmits<EditMiBoardStructElementViewEmits>()

const {
    description,
    apply,
} = useEditMiBoardStructElementView({ props, emits })
</script>
