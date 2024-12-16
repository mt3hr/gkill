<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>Rep追加</span>
                </v-col>
            </v-row>
        </v-card-title>
        <div>{{ account.user_id }}</div>
        <v-checkbox v-model="is_enable" :label="'有効'" />
        <v-checkbox v-model="use_to_write" :label="'書込先として使用'" />
        <v-checkbox v-model="is_execute_idf_when_reload" :label="'更新時にID自動割り当て'" />
        <v-text-field v-model="device" :label="'デバイス名'" />
        <v-select v-model="type" :items="rep_types" label="RepType" />
        <v-text-field v-model="file" :label="'PATH'" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="add_rep()">追加</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="emits('requested_close_dialog')">キャンセル</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddRepViewEmits } from './add-rep-view-emits'
import type { AddRepViewProps } from './add-rep-view-props'
import { Repository } from '@/classes/datas/config/repository';
import { GkillAPI } from '@/classes/api/gkill-api';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';

const props = defineProps<AddRepViewProps>()
const emits = defineEmits<AddRepViewEmits>()
const device: Ref<string> = ref("")
const type: Ref<string> = ref("")
const file: Ref<string> = ref("")
const use_to_write: Ref<boolean> = ref(false)
const is_execute_idf_when_reload: Ref<boolean> = ref(false)
const is_enable: Ref<boolean> = ref(true)

const rep_types: Ref<Array<string>> = ref([
    "kmemo",
    "urlog",
    "timeis",
    "mi",
    "nlog",
    "lantana",
    "tag",
    "text",
    "rekyou",
    "directory",
    "gpslog",
])

async function add_rep(): Promise<void> {
    const repository = new Repository()
    repository.id = GkillAPI.get_instance().generate_uuid()
    repository.device = device.value
    repository.user_id = props.account.user_id
    repository.type = type.value
    repository.file = file.value
    repository.use_to_write = use_to_write.value
    repository.is_execute_idf_when_reload = is_execute_idf_when_reload.value
    repository.is_enable = is_enable.value
    emits('requested_add_rep', repository)
    emits('requested_close_dialog')
}
</script>
