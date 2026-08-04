<template>
    <v-card class="pa-2" variant="flat">
        <v-card-title>
            {{ i18n.global.t("ADD_FOLDER_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="folder_name" :label="i18n.global.t('FOLDER_NAME_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
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
import type { AddNewFolderViewEmits } from './add-new-folder-view-emits'
import type { AddNewFolderViewProps } from './add-new-folder-view-props'
import { useAddNewFolderView } from '@/classes/use-add-new-folder-view'

const props = defineProps<AddNewFolderViewProps>()
const emits = defineEmits<AddNewFolderViewEmits>()

const {
    folder_name,
    emits_folder,
    reset_folder_name,
} = useAddNewFolderView({ props, emits })

defineExpose({ reset_folder_name })
</script>
