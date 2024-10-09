<template>
    <v-card @contextmenu.prevent="show_context_menu">
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <div>{{ kyou.typed_urlog?.title }}</div>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <img v-if="kyou.typed_urlog" class="urlog_favicon"
                    :src="'data:image;base64,' + (kyou.typed_urlog.favicon_image)" />
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <a v-if="kyou.typed_urlog" :href="kyou.typed_urlog.url">{{ kyou.typed_urlog.url }}</a>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <img v-if="kyou.typed_urlog" class="urlog_thumbnail"
                    :src="'data:image;base64,' + (kyou.typed_urlog.thumbnail_image)" />
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <div v-if="kyou.typed_urlog" class="urlog_description">{{ kyou.typed_urlog.description }}</div>
            </v-col>
        </v-row>
    </v-card>
    <URLogContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[kyou.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag"
        ref="context_menu" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import type { URLogViewProps } from './ur-log-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import URLogContextMenu from './ur-log-context-menu.vue'

const context_menu = ref<InstanceType<typeof URLogContextMenu> | null>(null);

const props = defineProps<URLogViewProps>()
const emits = defineEmits<KyouViewEmits>()

function show_context_menu(e: PointerEvent): void {
    context_menu.value?.show(e)
}
</script>

<style lang="css" scoped>
.urlog_favicon {
    height: 20px;
    min-height: 20px;
    max-height: 20px;
    width: 20px;
    min-width: 20px;
    max-width: 20px;
}

.urlog_thumbnail {
    height: 75px;
    min-height: 75px;
    max-height: 75px;
    width: 75px;
    min-width: 75px;
    max-width: 75px;
}

.urlog_description {
    height: 75px;
    min-height: 75px;
    max-height: 75px;
}
</style>