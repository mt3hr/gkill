<template>
    <TagStructElement :application_config="application_config" :folder_name="''" :gkill_api="gkill_api" :is_open="true"
        :struct_obj="application_config.parsed_tag_struct"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_update_is_check_when_inited_state="'//TODO'" @requested_update_is_force_hide_state="'//TODO'" />
    <AddNewTagStructElementDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
        :is_open="true" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_add_tag_struct_element="'//TODO'" />
    <EditTagStructElementDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
        :struct_obj="tag_struct" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_update_tag_struct_element="'//TODO'" />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditTagStructViewEmits } from './edit-tag-struct-view-emits'
import type { EditTagStructViewProps } from './edit-tag-struct-view-props'
import TagStructElement from './tag-struct-element.vue'
import { TagStruct } from '@/classes/datas/config/tag-struct'
import AddNewTagStructElementDialog from '../dialogs/add-new-tag-struct-element-dialog.vue'
import EditTagStructElementDialog from '../dialogs/edit-tag-struct-element-dialog.vue'

const props = defineProps<EditTagStructViewProps>()
const emits = defineEmits<EditTagStructViewEmits>()

const tag_struct: Ref<Array<TagStruct>> = ref(props.tag_struct.concat())
const tag_struct_root: Ref<Array<TagStruct>> = ref(props.tag_struct_root.concat())
</script>
