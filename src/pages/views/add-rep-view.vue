<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ADD_REP_TITLE") }}</span>
                </v-col>
            </v-row>
        </v-card-title>
        <div>{{ account.user_id }}</div>
        <v-checkbox v-model="is_enable" :label="i18n.global.t('ENABLE_TITLE')" />
        <v-checkbox v-model="use_to_write" :label="i18n.global.t('USE_TO_WRITE_TITLE')" />
        <v-checkbox v-model="is_execute_idf_when_reload" :label="i18n.global.t('USE_TO_WRITE_TITLE')" />
        <v-select :label="i18n.global.t('DEVICE_NAME_TITLE')" v-model="device" :items="devices" />
        <v-select v-model="type" :items="rep_types" :label="i18n.global.t('REP_TYPE_TITLE')" />
        <v-text-field v-model="file" :label="i18n.global.t('FILE_PATH_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="add_rep()" color="primary">{{ i18n.global.t("ADD_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ i18n.global.t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { AddRepViewEmits } from './add-rep-view-emits'
import type { AddRepViewProps } from './add-rep-view-props'
import { Repository } from '@/classes/datas/config/repository';

const props = defineProps<AddRepViewProps>()
const emits = defineEmits<AddRepViewEmits>()
const device: Ref<string> = ref("")
const type: Ref<string> = ref("")
const file: Ref<string> = ref("")
const use_to_write: Ref<boolean> = ref(false)
const is_execute_idf_when_reload: Ref<boolean> = ref(false)
const is_enable: Ref<boolean> = ref(true)

const devices: Ref<Array<string>> = ref((() => {
    const devices = Array<string>()
    for (let i = 0; i < props.server_configs.length; i++) {
        devices.push(props.server_configs[i].device)
    }
    return devices
})())

const rep_types: Ref<Array<string>> = ref([
    "kmemo",
    "kc",
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
    "git_commit_log",
    "notification",
])

async function add_rep(): Promise<void> {
    const repository = new Repository()
    repository.id = props.gkill_api.generate_uuid()
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
