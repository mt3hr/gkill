<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <a v-if="kyou.typed_idf_kyou && !kyou.typed_idf_kyou.is_image && !kyou.typed_idf_kyou.is_video && !kyou.typed_idf_kyou.is_audio"
            :href="kyou.typed_idf_kyou.file_url" @click="open_link">
            {{ kyou.typed_idf_kyou.file_name }}
        </a>
        <img v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_image"
            :src="kyou.typed_idf_kyou.file_url.concat(props.is_image_request_to_thumb_size ? '?thumb=400x400' : '')"
            loading="lazy" decording="async" fetchpriority="low" class="kyou_image" />
         <video v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_video" :src="kyou.typed_idf_kyou.file_url" preload="none"
            class="kyou_video" controls></video>
        <audio v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_audio" :src="kyou.typed_idf_kyou.file_url"
            class="kyou_audio" controls></audio>
        <IDFKyouContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" ref="context_menu"
            @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
            @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
            @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
            @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
            @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
            @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
            @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
            @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
            @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
            @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
            @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
            @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import IDFKyouContextMenu from './idf-kyou-context-menu.vue'
import { ref } from 'vue'
import type { IDFKyouProps } from './idf-kyou-props'
import type { KyouViewEmits } from './kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

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

<style lang="css" scoped>
.kyou_image {
    box-sizing: border-box;
    border: 1px solid gray;
    height: 200px;
    width: 200px;
    object-fit: cover;
}

.kyou_video {
    box-sizing: border-box;
    border: 1px solid gray;
    height: 200px;
    width: 200px;
    object-fit: cover;
}
</style>