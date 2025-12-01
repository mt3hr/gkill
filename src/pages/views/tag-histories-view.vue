<template>
    <TagView class="tag_history" v-for="tag in cloned_tag.attached_histories" :key="tag.update_time.getTime()"
        :application_config="application_config" :highlight_targets="highlight_targets" :gkill_api="gkill_api"
        :tag="tag" :kyou="kyou" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        :last_added_tag="last_added_tag"
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
        @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
        @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import type { TagHistoriesViewProps } from './tag-histories-view-props'
import { Tag } from '@/classes/datas/tag'
import { Text } from '@/classes/datas/text'
import { Notification } from '@/classes/datas/notification'
import TagView from './tag-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'

const props = defineProps<TagHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_tag: Ref<Tag> = ref(props.tag.clone())
watch(() => props.tag, () => {
    cloned_tag.value = props.tag.clone()
    cloned_tag.value.load_attached_histories()
})
cloned_tag.value.load_attached_histories()

</script>
<style lang="css">
.tag_history .tag_wrap {
    display: block;
    width: 400px;
}
</style>