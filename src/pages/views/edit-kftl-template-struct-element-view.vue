<template>
    <v-card>
        <v-card-title>
            {{ $t("EDIT_KFTL_TEMPLATE_STRUCT_ELEMENT_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="title" :label="$t('TEMPLATE_NAME_TITLE')" />
        <v-textarea v-model="template" :label="$t('TEMPLATE_CONTENT_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="apply" color="primary">{{ $t("APPLY_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ $t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditKFTLTemplateStructElementViewEmits } from './edit-kftl-template-struct-element-view-emits'
import type { EditKFTLTemplateStructElementViewProps } from './edit-kftl-template-struct-element-view-props'
import { KFTLTemplateStruct } from '@/classes/datas/config/kftl-template-struct';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';

const props = defineProps<EditKFTLTemplateStructElementViewProps>()
const emits = defineEmits<EditKFTLTemplateStructElementViewEmits>()

const title: Ref<string> = ref(props.struct_obj.title)
const template: Ref<string | null> = ref(props.struct_obj.template)

async function apply(): Promise<void> {
    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }
    if (gkill_info_res.messages && gkill_info_res.messages.length !== 0) {
        emits('received_messages', gkill_info_res.messages)
    }

    const kftl_template_struct = new KFTLTemplateStruct()
    kftl_template_struct.id = props.struct_obj.id
    kftl_template_struct.device = gkill_info_res.device
    kftl_template_struct.user_id = gkill_info_res.user_id
    kftl_template_struct.parent_folder_id = props.struct_obj.parent_folder_id
    kftl_template_struct.seq = props.struct_obj.seq
    kftl_template_struct.title = title.value
    kftl_template_struct.template = template.value
    emits('requested_update_kftl_template_struct', kftl_template_struct)
    emits('requested_close_dialog')
}
</script>
