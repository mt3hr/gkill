<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <a v-if="kyou.typed_idf_kyou && !kyou.typed_idf_kyou.is_image && !kyou.typed_idf_kyou.is_video && !kyou.typed_idf_kyou.is_audio"
            :href="kyou.typed_idf_kyou.file_url" @click="open_link">
            {{ kyou.typed_idf_kyou.file_name }}
        </a>
        <img v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_image"
            :src="buildMediaUrl(kyou.typed_idf_kyou.file_url, false)" loading="lazy" decording="async"
            fetchpriority="low" class="kyou_image" />
        <video v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_video" :src="kyou.typed_idf_kyou.file_url"
            preload="none" :poster="buildMediaUrl(kyou.typed_idf_kyou.file_url, true)" class="kyou_video"
            controls></video>
        <audio v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_audio" :src="kyou.typed_idf_kyou.file_url"
            class="kyou_audio" controls></audio>
        <IDFKyouContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" ref="context_menu"
            @deleted_kyou="crudRelayHandlers['deleted_kyou']"
            @deleted_tag="crudRelayHandlers['deleted_tag']"
            @deleted_text="crudRelayHandlers['deleted_text']"
            @deleted_notification="crudRelayHandlers['deleted_notification']"
            @registered_kyou="crudRelayHandlers['registered_kyou']"
            @registered_tag="crudRelayHandlers['registered_tag']"
            @registered_text="crudRelayHandlers['registered_text']"
            @registered_notification="crudRelayHandlers['registered_notification']"
            @updated_kyou="crudRelayHandlers['updated_kyou']"
            @updated_tag="crudRelayHandlers['updated_tag']"
            @updated_text="crudRelayHandlers['updated_text']"
            @updated_notification="crudRelayHandlers['updated_notification']"
            @received_errors="crudRelayHandlers['received_errors']"
            @received_messages="crudRelayHandlers['received_messages']"
            @requested_reload_kyou="crudRelayHandlers['requested_reload_kyou']"
            @requested_reload_list="crudRelayHandlers['requested_reload_list']"
            @requested_update_check_kyous="crudRelayHandlers['requested_update_check_kyous']"
            @requested_open_rykv_dialog="crudRelayHandlers['requested_open_rykv_dialog']" />
    </v-card>
</template>
<script setup lang="ts">
import IDFKyouContextMenu from './idf-kyou-context-menu.vue'
import type { IDFKyouProps } from './idf-kyou-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { useIDFKyouView } from '@/classes/use-idf-kyou-view'

const props = defineProps<IDFKyouProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    show_context_menu,
    open_link,
    buildMediaUrl,
    crudRelayHandlers,
} = useIDFKyouView({ props, emits })

defineExpose({ show_context_menu })
</script>