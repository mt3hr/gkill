<template>
    <!-- //TODO Emitsがまだ-->
    <!-- //TODO 削除ダイアログが存在しない-->
    <ApplicationConfigStructContextMenu :application_config="application_config" :folder_name="folder_name"
        :gkill_api="gkill_api" :is_open="is_open" :struct_obj="struct_obj" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import type { DeviceStructElementEmits } from './device-struct-element-emits'
import type { DeviceStructElementProps } from './device-struct-element-props'
import ApplicationConfigStructContextMenu from './application-config-struct-context-menu.vue'

const props = defineProps<DeviceStructElementProps>()
const emits = defineEmits<DeviceStructElementEmits>()
const cloned_is_open: Ref<boolean> = ref(props.is_open)
const struct_list: Ref<any> = ref(props.struct_obj)
const check_when_inited: Ref<boolean> = ref(false);//TODO
</script>
<style scoped></style>
