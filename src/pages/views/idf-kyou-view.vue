<template>
    <v-card @contextmenu.prevent="show_context_menu">
        <a v-if="kyou.typed_idf_kyou && !kyou.typed_idf_kyou.is_image" :href="kyou.typed_idf_kyou.file_url">
            {{ kyou.typed_idf_kyou.file_name }}
        </a>
        <img v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_image" :src="kyou.typed_idf_kyou.file_url" />
    </v-card>
    <IDFKyouContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[kyou.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag"
        ref="context_menu" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import IDFKyouContextMenu from './idf-kyou-context-menu.vue'
import type { IDFKyou } from '@/classes/datas/idf-kyou'
import { type Ref, ref } from 'vue'
import type { IDFKyouProps } from './idf-kyou-props'
import type { KyouViewEmits } from './kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'

const context_menu = ref<InstanceType<typeof IDFKyouContextMenu> | null>(null);

const props = defineProps<IDFKyouProps>()
const emits = defineEmits<KyouViewEmits>()

function show_context_menu(e: PointerEvent): void {
    context_menu.value?.show(e)
}
</script>
