<template>
    <ApplicationConfigStructContextMenu :application_config="application_config" :folder_name="''"
        :gkill_api="gkill_api" :is_open="true" :struct_obj="struct_obj" @requested_show_delete_dialog="''"
        @requested_show_edit_dialog="''" />
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { RepStructElementEmits } from './rep-struct-element-emits'
import type { RepStructElementProps } from './rep-struct-element-props'

defineProps<RepStructElementProps>()
defineEmits<RepStructElementEmits>()
</script>
