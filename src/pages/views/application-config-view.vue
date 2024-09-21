<template>
    <!-- //TODO 実装 -->
    <EditDeviceStructDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
        :struct_obj="cloned_application_config.parsed_device_struct"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct"
        @requested_update_device_struct_element="async () => emits('requested_reload_application_config', await get_application_config())" />
    <EditKFTLTemplateDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config="application_config" :gkill_api="gkill_api"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct"
        @requested_reload_application_config="(application_config) => emits('requested_reload_application_config', application_config)" />
    <EditRepStructDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config="application_config" :gkill_api="gkill_api"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct"
        @requested_reload_application_config="(application_config) => emits('requested_reload_application_config', application_config)" />
    <EditRepTypeDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config="application_config" :gkill_api="gkill_api"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct"
        @requested_reload_application_config="(application_config) => emits('requested_reload_application_config', application_config)" />
    <EditTagStructDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config="application_config" :gkill_api="gkill_api"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct"
        @requested_reload_application_config="(application_config) => emits('requested_reload_application_config', application_config)" />
</template>
<script setup lang="ts">
import type { GkillError } from '@/classes/api/gkill-error'
import { type Ref, ref } from 'vue'

import EditDeviceStructDialog from '../dialogs/edit-device-struct-dialog.vue'
import EditKFTLTemplateDialog from '../dialogs/edit-kftl-template-dialog.vue'
import EditRepStructDialog from '../dialogs/edit-rep-struct-dialog.vue'
import EditRepTypeDialog from '../dialogs/edit-rep-type-dialog.vue'
import EditTagStructDialog from '../dialogs/edit-tag-struct-dialog.vue'

import type { ApplicationConfigViewEmits } from './application-config-view-emits'
import type { ApplicationConfigViewProps } from './application-config-view-props'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'

const props = defineProps<ApplicationConfigViewProps>()
const emits = defineEmits<ApplicationConfigViewEmits>()

const cloned_application_config = ref(await props.application_config.clone())
const google_map_api_key: Ref<string> = ref(cloned_application_config.value.google_map_api_key)
const number_of_rykv_columns: Ref<Number> = ref(cloned_application_config.value.rykv_image_list_column_number)
const default_board_name_of_mi: Ref<string> = ref(cloned_application_config.value.mi_default_board)
const is_enable_browser_cache: Ref<boolean> = ref(cloned_application_config.value.enable_browser_cache)
const is_enable_hot_reload_rykv: Ref<boolean> = ref(cloned_application_config.value.rykv_hot_reload)

async function apply_application_config(): Promise<GkillError> {
    throw new Error('Not implemented')
}

async function get_application_config(): Promise<ApplicationConfig> {
    const req = new GetApplicationConfigRequest()
    req.session_id = "" //TODO セッションIDどこ
    const res = await props.gkill_api.get_application_config(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
    }
    return res.application_config
}
</script>
