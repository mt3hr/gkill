<template>
    <RepStructElement :application_config="application_config" :folder_name="''" :gkill_api="gkill_api" :is_open="true"
        :struct_obj="application_config.rep_struct" />
</template>
<script lang="ts" setup>
import { ref } from 'vue'
import type { EditRepStructViewEmits } from './edit-rep-struct-view-emits'
import type { EditRepStructViewProps } from './edit-rep-struct-view-props'
import RepStructElement from './rep-struct-element.vue'
import TagStructElement from './tag-struct-element.vue'
import AddNewRepStructElementDialog from '../dialogs/add-new-rep-struct-element-dialog.vue'
import EditRepStructElementDialog from '../dialogs/edit-rep-struct-element-dialog.vue'

const props = defineProps<EditRepStructViewProps>()
const emits = defineEmits<EditRepStructViewEmits>()
</script>
