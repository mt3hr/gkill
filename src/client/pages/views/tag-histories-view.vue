<template>
    <TagView class="tag_history" v-for="tag in cloned_tag.attached_histories" :key="tag.update_time.getTime()"
        :application_config="application_config" :highlight_targets="highlight_targets" :gkill_api="gkill_api"
        :tag="tag" :kyou="kyou" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="crudRelayHandlers" />
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from './kyou-view-emits'
import type { TagHistoriesViewProps } from './tag-histories-view-props'
import TagView from './tag-view.vue'
import { useTagHistoriesView } from '@/classes/use-tag-histories-view'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

const props = defineProps<TagHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()

const crudRelayHandlers = build_kyou_view_relay(emits)

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
