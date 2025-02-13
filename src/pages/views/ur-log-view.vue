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
            @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
            @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
            @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
            @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
            @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
            @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
            @registered_text="(registered_text) => emits('registered_text', registered_text)"
            @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
            @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
            @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
            @updated_text="(updated_text) => emits('updated_text', updated_text)"
            @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    </v-card>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import type { URLogViewProps } from './ur-log-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import URLogContextMenu from './ur-log-context-menu.vue'

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