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
                        :src="kyou.typed_urlog.thumbnail_image === '' ? '/noimage.png' : 'data:image;base64,' + (kyou.typed_urlog.favicon_image)" />
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
                        :src="kyou.typed_urlog.thumbnail_image === '' ? '/noimage.png' : 'data:image;base64,' + (kyou.typed_urlog.thumbnail_image)" />
                </td>
                <td>
                    <div v-if="kyou.typed_urlog" class="urlog_description">{{ kyou.typed_urlog.description }}</div>
                </td>
            </tr>
        </table>
        <URLogContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag" ref="context_menu"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
            @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
            @deleted_text="(...deleted_text :any[]) => emits('deleted_text', deleted_text[0] as Text)"
            @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
            @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
            @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
            @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
            @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
            @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
            @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
            @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
            @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
    </v-card>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import type { URLogViewProps } from './ur-log-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import URLogContextMenu from './ur-log-context-menu.vue'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'

const context_menu = ref<InstanceType<typeof URLogContextMenu> | null>(null);

const props = defineProps<URLogViewProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show_context_menu })

function show_context_menu(e: PointerEvent): void {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}

function open_urlog_link(): void {
    const url = props.kyou.typed_urlog?.url
    if (url) {
        window.open(url, "_blank")
    }
}
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