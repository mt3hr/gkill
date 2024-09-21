<template>
    <!-- //TODO 削除確認ダイアログがない -->
    <ApplicationConfigStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :folder_name="folder_name" :is_open="is_open" :struct_obj="struct_obj" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import ApplicationConfigStructContextMenu from './application-config-struct-context-menu.vue'
import type { TagStructElementEmits } from './tag-struct-element-emits'
import type { TagStructElementProps } from './tag-struct-element-props'

const props = defineProps<TagStructElementProps>()
const emits = defineEmits<TagStructElementEmits>()
const cloned_is_open: Ref<boolean> = ref(props.is_open)
const struct_list: Ref<any> = ref(props.struct_obj)
const is_check_when_inited: Ref<boolean> = ref(true)//TODO
const is_force_hide: Ref<boolean> = ref(false) //TODO
</script>
