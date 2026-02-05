<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("EDIT_REP_TYPE_TITLE") }}
        </v-card-title>
        <p>{{ struct_obj.rep_type_name }}</p>
        <v-checkbox v-model="check_when_inited" hide-detail :label="i18n.global.t('CHECK_WHEN_INITED_TITLE')" />
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
import type { EditRepTypeStructElementViewEmits } from './edit-rep-type-struct-element-view-emits'
import type { EditRepTypeStructElementViewProps } from './edit-rep-type-struct-element-view-props'
import { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data';

const props = defineProps<EditRepTypeStructElementViewProps>()
const emits = defineEmits<EditRepTypeStructElementViewEmits>()

const check_when_inited: Ref<boolean> = ref(props.struct_obj.check_when_inited)

async function apply(): Promise<void> {
    const rep_type_struct = new RepTypeStructElementData()
    rep_type_struct.id = props.struct_obj.id
    rep_type_struct.check_when_inited = check_when_inited.value
    rep_type_struct.children = null
    rep_type_struct.indeterminate = false
    rep_type_struct.key = props.struct_obj.rep_type_name
    rep_type_struct.rep_type_name = props.struct_obj.rep_type_name
    rep_type_struct.name = props.struct_obj.rep_type_name

    emits('requested_update_rep_type_struct', rep_type_struct)
    emits('requested_close_dialog')
}
</script>
