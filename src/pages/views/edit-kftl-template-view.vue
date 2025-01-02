<template>
    <EditKftlTemplateStructElementDialog :application_config="application_config" :folder_name="''"
        :gkill_api="gkill_api" :is_open="true" :struct_obj="application_config.kftl_template_struct" />
</template>
<script lang="ts" setup>
import type { EditKFTLTemplateViewEmits } from './edit-kftl-template-view-emits'
import type { EditKFTLTemplateViewProps } from './edit-kftl-template-view-props'
import EditKftlTemplateStructElementDialog from '../dialogs/edit-kftl-template-struct-element-dialog.vue'

defineProps<EditKFTLTemplateViewProps>()
defineEmits<EditKFTLTemplateViewEmits>()
</script>
