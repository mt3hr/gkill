<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <div v-if="kyou.typed_kmemo" class="kmemo_text_content">{{ kyou.typed_kmemo.content }}</div>
        <KmemoContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="[kyou.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
            ref="context_menu" />
    </v-card>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import KmemoContextMenu from './kmemo-context-menu.vue'
import type { Kmemo } from '@/classes/datas/kmemo'
import type { KmemoViewProps } from './kmemo-view-props.ts'
import type { KyouViewEmits } from './kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'

const props = defineProps<KmemoViewProps>()
const emits = defineEmits<KyouViewEmits>()

const context_menu = ref<InstanceType<typeof KmemoContextMenu> | null>(null);

async function show_context_menu(e: PointerEvent): Promise<void> {
    context_menu.value?.show(e)
}

</script>

<style lang="css" scoped>
.kmemo_text_content {
    white-space: pre-line;
}
</style>