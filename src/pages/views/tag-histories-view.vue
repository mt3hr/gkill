<template>
    <TagView v-for="tag, index in cloned_tag.attached_histories" :application_config="application_config"
        :highlight_targets="highlight_targets" :gkill_api="gkill_api" :tag="tag" :kyou="kyou"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :last_added_tag="last_added_tag"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
</template>
<script lang="ts" setup>
import { type Ref, computed, nextTick, ref, watch } from 'vue'
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
