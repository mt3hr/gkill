<template>
    <TagView class="tag_history" v-for="tag in cloned_tag.attached_histories" :key="tag.update_time.getTime()"
        :application_config="application_config" :highlight_targets="highlight_targets" :gkill_api="gkill_api"
        :tag="tag" :kyou="kyou" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        :last_added_tag="last_added_tag" @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
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
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import type { TagHistoriesViewProps } from './tag-histories-view-props'
import { Tag } from '@/classes/datas/tag'
import TagView from './tag-view.vue'

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