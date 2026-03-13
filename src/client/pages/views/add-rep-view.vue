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
import type { AddRepViewEmits } from './add-rep-view-emits'
import type { AddRepViewProps } from './add-rep-view-props'
import { useAddRepView } from '@/classes/use-add-rep-view'

const props = defineProps<AddRepViewProps>()
const emits = defineEmits<AddRepViewEmits>()

const {
    device,
    type,
    file,
    use_to_write,
    is_execute_idf_when_reload,
    is_enable,
    devices,
    rep_types,
    add_rep,
} = useAddRepView({ props, emits })
</script>
