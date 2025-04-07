<template>
    <ApplicationConfigStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :folder_name="folder_name" :is_open="is_open" :struct_obj="struct_obj" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import type { TagStructElementEmits } from './tag-struct-element-emits'
import type { TagStructElementProps } from './tag-struct-element-props'
import type { ApplicationConfig } from '@/classes/datas/config/application-config';

const props = defineProps<TagStructElementProps>()
defineEmits<TagStructElementEmits>()
defineExpose({ reload_cloned_application_config })

const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
}
</script>
