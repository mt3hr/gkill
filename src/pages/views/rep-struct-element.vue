<template>
    <ApplicationConfigStructContextMenu :application_config="application_config" :folder_name="''"
        :gkill_api="gkill_api" :is_open="true" :struct_obj="struct_obj" @requested_show_delete_dialog="'//TODO'"
        @requested_show_edit_dialog="'//TODO'" />
</template>
<script setup lang="ts">
import type { RepStructElementEmits } from './rep-struct-element-emits'
import type { RepStructElementProps } from './rep-struct-element-props'
import ApplicationConfigStructContextMenu from './application-config-struct-context-menu.vue'

defineProps<RepStructElementProps>()
defineEmits<RepStructElementEmits>()
</script>
