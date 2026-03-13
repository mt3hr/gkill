<template>
    <ApplicationConfigStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :folder_name="folder_name" :is_open="is_open" :struct_obj="struct_obj" />
</template>
<script setup lang="ts">
import { useTagStructElement } from '@/classes/use-tag-struct-element'
import type { TagStructElementEmits } from './tag-struct-element-emits'
import type { TagStructElementProps } from './tag-struct-element-props'

const props = defineProps<TagStructElementProps>()
defineEmits<TagStructElementEmits>()

const {
    cloned_application_config,
    reload_cloned_application_config,
} = useTagStructElement({ props })

defineExpose({ reload_cloned_application_config })
</script>
