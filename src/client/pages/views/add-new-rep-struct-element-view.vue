<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("ADD_REP_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="rep_name" :label="i18n.global.t('REP_TITLE')" />
        <v-checkbox v-model="check_when_inited" hide-detail :label="i18n.global.t('IS_FORCE_HIDE_TITLE')" />
        <v-checkbox v-model="ignore_check_rep_rykv" hide-detail :label="i18n.global.t('IGNORE_CHECK_REP_RYKV_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="emits_rep_name">{{ i18n.global.t("ADD_TITLE") }}</v-btn>
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
import type { AddNewRepStructElementViewEmits } from './add-new-rep-struct-element-view-emits'
import type { AddNewRepStructElementViewProps } from './add-new-rep-struct-element-view-props'
import { useAddNewRepStructElementView } from '@/classes/use-add-new-rep-struct-element-view'

const props = defineProps<AddNewRepStructElementViewProps>()
const emits = defineEmits<AddNewRepStructElementViewEmits>()

const {
    rep_name,
    check_when_inited,
    ignore_check_rep_rykv,
    emits_rep_name,
    reset_rep_name,
} = useAddNewRepStructElementView({ props, emits })

defineExpose({ reset_rep_name })
</script>
