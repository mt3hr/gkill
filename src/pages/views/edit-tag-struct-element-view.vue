<template>
    <v-card>
        <v-card-title>
            {{ $t('EDIT_TAG_STRUCT_ELEMENT') }}
        </v-card-title>
        <p>{{ struct_obj.tag_name }}</p>
        <v-checkbox v-model="check_when_inited" hide-detail :label="$t('CHECK_WHEN_INITED_TITLE')" />
        <v-checkbox v-model="is_force_hide" hide-detail :label="$t('IS_FORCE_HIDE_TITLE')" />
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
import type { EditTagStructElementViewEmits } from './edit-tag-struct-element-view-emits'
import type { EditTagStructElementViewProps } from './edit-tag-struct-element-view-props'
import { TagStruct } from '@/classes/datas/config/tag-struct';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';

const props = defineProps<EditTagStructElementViewProps>()
const emits = defineEmits<EditTagStructElementViewEmits>()

const check_when_inited: Ref<boolean> = ref(props.struct_obj.check_when_inited)
const is_force_hide: Ref<boolean> = ref(props.struct_obj.is_force_hide)

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

    const tag_struct = new TagStruct()
    tag_struct.id = props.struct_obj.id
    tag_struct.device = gkill_info_res.device
    tag_struct.user_id = gkill_info_res.user_id
    tag_struct.tag_name = props.struct_obj.tag_name
    tag_struct.parent_folder_id = props.struct_obj.parent_folder_id
    tag_struct.check_when_inited = check_when_inited.value
    tag_struct.is_force_hide = is_force_hide.value
    tag_struct.seq = props.struct_obj.seq
    emits('requested_update_tag_struct', tag_struct)
    emits('requested_close_dialog')
}
</script>
