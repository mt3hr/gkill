<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("ADD_FOLDER_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="folder_name" :label="i18n.global.t('FOLDER_NAME_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="emits_folder">{{ i18n.global.t("ADD_TITLE") }}</v-btn>
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
import type { AddNewFoloderViewEmits } from './add-new-foloder-view-emits'
import type { AddNewFoloderViewProps } from './add-new-foloder-view-props'
import { useAddNewFoloderView } from '@/classes/use-add-new-foloder-view'

const props = defineProps<AddNewFoloderViewProps>()
const emits = defineEmits<AddNewFoloderViewEmits>()

const {
    folder_name,
    emits_folder,
    reset_folder_name,
} = useAddNewFoloderView({ props, emits })

defineExpose({ reset_folder_name })
</script>
