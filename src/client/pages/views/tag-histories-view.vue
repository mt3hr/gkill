<template>
    <TagView class="tag_history" v-for="tag in cloned_tag.attached_histories" :key="tag.update_time.getTime()"
        :application_config="application_config" :highlight_targets="highlight_targets" :gkill_api="gkill_api"
        :tag="tag" :kyou="kyou" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
        @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
        @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)" />
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from './kyou-view-emits'
import type { TagHistoriesViewProps } from './tag-histories-view-props'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import TagView from './tag-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import { useTagHistoriesView } from '@/classes/use-tag-histories-view'

const props = defineProps<TagHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    cloned_tag,
} = useTagHistoriesView({ props, emits })
</script>
<style lang="css">
.tag_history .tag_wrap {
    display: block;
    width: 400px;
}
</style>
