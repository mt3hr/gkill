<template>
    <ApplicationConfigStructContextMenu :application_config="application_config" :folder_name="''"
        :gkill_api="gkill_api" :is_open="true" :struct_obj="struct_obj" @requested_show_delete_dialog="'//TODO'"
        @requested_show_edit_dialog="'//TODO'" />
</template>
<script setup lang="ts">
import type { RepStructElementEmits } from './rep-struct-element-emits'
import type { RepStructElementProps } from './rep-struct-element-props'
import ApplicationConfigStructContextMenu from './application-config-struct-context-menu.vue'
import { type Ref, ref } from 'vue'

const props = defineProps<RepStructElementProps>()
const emits = defineEmits<RepStructElementEmits>()
const cloned_is_open: Ref<boolean> = ref(props.is_open)
const struct_list: Ref<any> = ref(props.struct_obj)//TODO
const check_when_inited: Ref<boolean> = ref(false)//TODO
const ignore_check_rep_rykv: Ref<boolean> = ref(false)//TODO
</script>
