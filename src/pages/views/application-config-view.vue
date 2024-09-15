<template>
    <EditDeviceStructDialog />
    <EditKFTLTemplateDialog />
    <EditRepStructDialog />
    <EditRepTypeDialog />
    <EditTagStructDialog />
</template>
<script setup lang="ts">
import type { GkillError } from '@/classes/api/gkill-error';
import { type Ref, ref } from 'vue';

import EditDeviceStructDialog from '../dialogs/edit-device-struct-dialog.vue';
import EditKFTLTemplateDialog from '../dialogs/edit-kftl-template-dialog.vue';
import EditRepStructDialog from '../dialogs/edit-rep-struct-dialog.vue';
import EditRepTypeDialog from '../dialogs/edit-rep-type-dialog.vue';
import EditTagStructDialog from '../dialogs/edit-tag-struct-dialog.vue';

import type { ApplicationConfigViewEmits } from './application-config-view-emits';
import type { ApplicationConfigViewProps } from './application-config-view-props';

const props = defineProps<ApplicationConfigViewProps>();
const emits = defineEmits<ApplicationConfigViewEmits>();

const cloned_application_config = ref(await props.application_config.clone());
const google_map_api_key: Ref<string> = ref(cloned_application_config.value.google_map_api_key);
const number_of_rykv_columns: Ref<Number> = ref(cloned_application_config.value.rykv_image_list_column_number);
const default_board_name_of_mi: Ref<string> = ref(cloned_application_config.value.mi_default_board);
const is_enable_browser_cache: Ref<boolean> = ref(cloned_application_config.value.enable_browser_cache);
const is_enable_hot_reload_rykv: Ref<boolean> = ref(cloned_application_config.value.rykv_hot_reload);

async function apply_application_config(): Promise<GkillError> {
    throw new Error('Not implemented');
}
</script>
