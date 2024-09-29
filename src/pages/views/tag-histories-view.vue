<template>
    <TagView v-for="tag, index in tag.attached_histories" :application_config="application_config"
        :gkill_api="gkill_api" :tag="tag" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
</template>
<script lang="ts" setup>
import { type Ref, computed, ref } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import type { TagHistoriesViewProps } from './tag-histories-view-props'
import { Tag } from '@/classes/datas/tag'
import TagView from './tag-view.vue'

const props = defineProps<TagHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()
</script>
