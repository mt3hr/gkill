<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("ADD_REP_TYPE_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="rep_type_name" :label="i18n.global.t('REP_TYPE_TITLE')" />
        <v-checkbox v-model="check_when_inited" hide-detail :label="i18n.global.t('CHECK_WHEN_INITED_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="emits_rep_type_name">{{ i18n.global.t("ADD_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{
                        i18n.global.t("CANCEL_TITLE")
                    }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AddNewRepTypeStructElementViewEmits } from './add-new-rep-type-struct-element-view-emits'
import type { AddNewRepTypeStructElementViewProps } from './add-new-rep-type-struct-element-view-props'
import { useAddNewRepTypeStructElementView } from '@/classes/use-add-new-rep-type-struct-element-view'

const props = defineProps<AddNewRepTypeStructElementViewProps>()
const emits = defineEmits<AddNewRepTypeStructElementViewEmits>()

const {
    rep_type_name,
    check_when_inited,
    emits_rep_type_name,
    reset_rep_type_name,
} = useAddNewRepTypeStructElementView({ props, emits })

defineExpose({ reset_rep_type_name })
</script>
