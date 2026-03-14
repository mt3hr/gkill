<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <table>
            <tr>
                <td>
                    <div class="urlog_title">{{ kyou.typed_urlog?.title }}</div>
                </td>
            </tr>
        </table>
        <table>
            <tr>
                <td>
                    <img v-if="kyou.typed_urlog" class="urlog_favicon"
                        :src="kyou.typed_urlog.thumbnail_image === '' ? '/noimage.png' : base64ToDataURI(kyou.typed_urlog.favicon_image)" />
                </td>
                <td>
                    <a v-if="kyou.typed_urlog" :href="kyou.typed_urlog.url" target="_blank" @click="open_urlog_link"
                        style="white-space: nowrap;">{{
                            kyou.typed_urlog.url }}</a>
                </td>
            </tr>
        </table>
        <table>
            <tr>
                <td>
                    <img v-if="kyou.typed_urlog" class="urlog_thumbnail"
                        :src="kyou.typed_urlog.thumbnail_image === '' ? '/noimage.png' : base64ToDataURI(kyou.typed_urlog.thumbnail_image)" />
                </td>
                <td>
                    <div v-if="kyou.typed_urlog" class="urlog_description">{{ kyou.typed_urlog.description }}</div>
                </td>
            </tr>
        </table>
        <URLogContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" ref="context_menu"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
import type { URLogViewProps } from './ur-log-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import URLogContextMenu from './ur-log-context-menu.vue'
import { useURLogView } from '@/classes/use-ur-log-view'

const props = defineProps<URLogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    show_context_menu,
    open_urlog_link,
    crudRelayHandlers,
} = useURLogView({ props, emits })

function base64ToDataURI(base64: string): string {
    if (base64.startsWith('/9j/')) return 'data:image/jpeg;base64,' + base64
    if (base64.startsWith('iVBOR')) return 'data:image/png;base64,' + base64
    if (base64.startsWith('R0lG')) return 'data:image/gif;base64,' + base64
    if (base64.startsWith('UklG')) return 'data:image/webp;base64,' + base64
    return 'data:image/png;base64,' + base64
}

defineExpose({ show_context_menu })
</script>

<style lang="css" scoped>
.urlog_title {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.urlog_favicon {
    height: 20px;
    min-height: 20px;
    max-height: 20px;
    width: 20px;
    min-width: 20px;
    max-width: 20px;
    object-fit: cover;
}

.urlog_thumbnail {
    height: 75px;
    min-height: 75px;
    max-height: 75px;
    width: 75px;
    min-width: 75px;
    max-width: 75px;
    object-fit: cover;
}

.urlog_description {
    height: 75px;
    min-height: 75px;
    max-height: 75px;
    overflow: hidden;
    text-overflow: ellipsis;
}
</style>