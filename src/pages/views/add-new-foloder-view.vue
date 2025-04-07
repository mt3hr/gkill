<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ $t("ADD_FOLDER_TITLE") }}
        </v-card-title>
        <v-text-field class="input" type="text" v-model="folder_name" :label="$t('FOLDER_NAME_TITLE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="emits_folder">{{ $t("ADD_TITLE") }}</v-btn>
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
import { ref, type Ref } from 'vue';
import type { AddNewFoloderViewEmits } from './add-new-foloder-view-emits'
import type { AddNewFoloderViewProps } from './add-new-foloder-view-props'
import { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data';
import { GkillError } from '@/classes/api/gkill-error';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<AddNewFoloderViewProps>()
const emits = defineEmits<AddNewFoloderViewEmits>()

defineExpose({ reset_folder_name })

const folder_name: Ref<string> = ref("")

function emits_folder(): void {
    if (folder_name.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.folder_name_is_blank
        error.error_message = t("FOLDER_NAME_IS_BLANK_MESSAGE")
        emits('received_errors', [error])
        return
    }

    const folder_struct_element = new FolderStructElementData()
    folder_struct_element.id = props.gkill_api.generate_uuid()
    folder_struct_element.folder_name = folder_name.value
    emits('requested_add_new_folder', folder_struct_element)
    emits('requested_close_dialog')
}

function reset_folder_name(): void {
    folder_name.value = ""
}
</script>
