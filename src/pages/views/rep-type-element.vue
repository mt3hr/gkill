<template>
    <!-- //TODO 実装-->
    <!-- //TODO emitsまだひろえてない-->
    <ApplicationConfigStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :folder_name="folder_name" :is_open="is_open" :struct_obj="struct_obj"
        />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import ApplicationConfigStructContextMenu from './application-config-struct-context-menu.vue'
import type { RepTypeElementEmits } from './rep-type-element-emits'
import type { RepTypeElementProps } from './rep-type-element-props'

const props = defineProps<RepTypeElementProps>()
const emits = defineEmits<RepTypeElementEmits>()
const cloned_is_open: Ref<boolean> = ref(props.is_open)
const struct_list: Ref<any> = ref(props.struct_obj)
const check_when_inited: Ref<boolean> = ref(false); //TODO
</script>
