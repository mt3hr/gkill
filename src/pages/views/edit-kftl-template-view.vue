<template>
    <EditKftlTemplateStructElementDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
        :is_open="true" :struct_obj="application_config.kftl_template" />
</template>
<script lang="ts" setup>
import type { EditKFTLTemplateViewEmits } from './edit-kftl-template-view-emits'
import type { EditKFTLTemplateViewProps } from './edit-kftl-template-view-props'
import KFTLTemplateElement from './kftl-template-element.vue'
import TagStructElement from './tag-struct-element.vue'
import AddNewKftlTemplateStructElementDialog from '../dialogs/add-new-kftl-template-struct-element-dialog.vue'
import EditKftlTemplateStructElementDialog from '../dialogs/edit-kftl-template-struct-element-dialog.vue'
import { ref } from 'vue'

const props = defineProps<EditKFTLTemplateViewProps>()
const emits = defineEmits<EditKFTLTemplateViewEmits>()
</script>
