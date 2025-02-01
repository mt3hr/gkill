<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <a v-if="kyou.typed_idf_kyou && !kyou.typed_idf_kyou.is_image && !kyou.typed_idf_kyou.is_video && !kyou.typed_idf_kyou.is_audio"
            :href="kyou.typed_idf_kyou.file_url" @click="open_link">
            {{ kyou.typed_idf_kyou.file_name }}
        </a>
        <img v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_image" :src="kyou.typed_idf_kyou.file_url"
            class="kyou_image" />
        <video v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_video" :src="kyou.typed_idf_kyou.file_url"
            class="kyou_video" controls></video>
        <audio v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_audio" :src="kyou.typed_idf_kyou.file_url"
            class="kyou_audio"></audio>
        <IDFKyouContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" ref="context_menu"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    </v-card>
</template>
<script setup lang="ts">
import IDFKyouContextMenu from './idf-kyou-context-menu.vue'
import { ref } from 'vue'
import type { IDFKyouProps } from './idf-kyou-props'
import type { KyouViewEmits } from './kyou-view-emits'

const context_menu = ref<InstanceType<typeof IDFKyouContextMenu> | null>(null);

const props = defineProps<IDFKyouProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show_context_menu })

function show_context_menu(e: PointerEvent): void {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}

function open_link(): void {
    const url = props.kyou.typed_idf_kyou?.file_url
    if (url) {
        window.open(url, "_blank")
    }
}
</script>

<style lang="css">
.kyou_image {
    border: 1px solid gray;
    height: 198px;
    width: 198px;
    object-fit: cover;
}

.kyou_video {
    border: 1px solid gray;
    height: 198px;
    width: 198px;
    object-fit: cover;
}

.kyou_audio {
    border: 1px solid gray;
    height: 198px;
    width: 198px;
    object-fit: cover;
}
</style>