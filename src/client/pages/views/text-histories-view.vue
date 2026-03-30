<template>
    <TextView class="text_history" v-for="text in cloned_text.attached_histories" :key="text.id"
        :application_config="application_config" :gkill_api="gkill_api" :text="text" :kyou="kyou"
        :highlight_targets="highlight_targets"
        @deleted_kyou="(deleted_kyou: Kyou) => emits('deleted_kyou', deleted_kyou)"
        @deleted_tag="(deleted_tag: Tag) => emits('deleted_tag', deleted_tag)"
        @deleted_text="(deleted_text: Text) => emits('deleted_text', deleted_text)"
        @deleted_notification="(deleted_notification: Notification) => emits('deleted_notification', deleted_notification)"
        @registered_kyou="(registered_kyou: Kyou) => emits('registered_kyou', registered_kyou)"
        @registered_tag="(registered_tag: Tag) => emits('registered_tag', registered_tag)"
        @registered_text="(registered_text: Text) => emits('registered_text', registered_text)"
        @registered_notification="(registered_notification: Notification) => emits('registered_notification', registered_notification)"
        @updated_kyou="(updated_kyou: Kyou) => emits('updated_kyou', updated_kyou)"
        @updated_tag="(updated_tag: Tag) => emits('updated_tag', updated_tag)"
        @updated_text="(updated_text: Text) => emits('updated_text', updated_text)"
        @updated_notification="(updated_notification: Notification) => emits('updated_notification', updated_notification)"
        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)" :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog" @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')" />
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from './kyou-view-emits'
import type { TextHistoriesViewProps } from './text-histories-view-props'
import TextView from './text-view.vue'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import { useTextHistoriesView } from '@/classes/use-text-histories-view'

const props = defineProps<TextHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    cloned_text,
} = useTextHistoriesView({ props, emits })
</script>
<style lang="css">
.text_history .highlighted_text,
.text_history .text {
    width: 400px;
}
</style>
