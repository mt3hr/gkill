<template>
    <MiContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="cloned_kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script lang="ts" setup>
import type { Mi } from '@/classes/datas/mi'
import { type Ref, ref } from 'vue'
import MiContextMenu from './mi-context-menu.vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { miKyouViewProps } from './mi-kyou-view-props'
import type { miKyouViewEmits } from './mi-kyou-view-emits'

const props = defineProps<miKyouViewProps>()
const emits = defineEmits<miKyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone())
const cloned_mi: Ref<Mi> = ref(cloned_kyou.value.typed_mi)
</script>