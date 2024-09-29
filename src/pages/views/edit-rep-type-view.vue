<template>
    <RepTypeElement :application_config="application_config" :folder_name="''" :gkill_api="gkill_api" :is_open="true"
        :struct_obj="application_config.rep_type_struct" />
</template>
<script lang="ts" setup>
import type { EditRepTypeViewEmits } from './edit-rep-type-view-emits'
import type { EditRepTypeViewProps } from './edit-rep-type-view-props'
import RepTypeElement from './rep-type-element.vue'
import TagStructElement from './tag-struct-element.vue'
import AddNewRepTypeStructElementDialog from '../dialogs/add-new-rep-type-struct-element-dialog.vue'
import EditRepTypeStructElementDialog from '../dialogs/edit-rep-type-struct-element-dialog.vue'
import { ref } from 'vue'

const props = defineProps<EditRepTypeViewProps>()
const emits = defineEmits<EditRepTypeViewEmits>()
</script>
