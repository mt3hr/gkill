<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("EDIT_REP_STRUCT_TITLE") }}
        </v-card-title>
        <p>{{ struct_obj.rep_name }}</p>
        <v-checkbox v-model="check_when_inited" hide-detail :label="i18n.global.t('CHECK_WHEN_INITED_TITLE')" />
        <v-checkbox v-model="ignore_check_rep_rykv" hide-detail :label="i18n.global.t('IGNORE_CHECK_REP_RYKV_TITLE')" />
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
import { type Ref, ref } from 'vue';
import type { EditRepStructElementViewEmits } from './edit-rep-struct-element-view-emits'
import type { EditRepStructElementViewProps } from './edit-rep-struct-element-view-props'
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data';

const props = defineProps<EditRepStructElementViewProps>()
const emits = defineEmits<EditRepStructElementViewEmits>()

const check_when_inited: Ref<boolean> = ref(props.struct_obj.check_when_inited)
const ignore_check_rep_rykv: Ref<boolean> = ref(props.struct_obj.ignore_check_rep_rykv)

async function apply(): Promise<void> {
    const rep_struct = new RepStructElementData()
    rep_struct.id = props.struct_obj.id
    rep_struct.key = props.struct_obj.rep_name
    rep_struct.rep_name = props.struct_obj.rep_name
    rep_struct.name = props.struct_obj.rep_name
    rep_struct.check_when_inited = check_when_inited.value
    rep_struct.ignore_check_rep_rykv = ignore_check_rep_rykv.value
    rep_struct.children = null
    rep_struct.indeterminate = false
    emits('requested_update_rep_struct', rep_struct)
    emits('requested_close_dialog')
}
</script>
