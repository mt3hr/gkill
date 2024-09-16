<template>
    <TagView :application_config="application_config" :gkill_api="gkill_api" :tag="tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
</template>
<script lang="ts" setup>
import { type Ref, computed, ref } from 'vue';
import type { KyouViewEmits } from './kyou-view-emits';
import type { TagHistoriesViewProps } from './tag-histories-view-props';
import { Tag } from '@/classes/datas/tag';
import TagView from './tag-view.vue';

const props = defineProps<TagHistoriesViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_tag: Ref<Tag> = ref(await props.tag.clone());
const cloned_tag_histories = computed(async () => {
    const errors = await cloned_tag.value.load_attached_histories()
    if (errors && errors.length !== 0) {
        emits('received_errors', errors)
        return
    }
    return cloned_tag.value.attached_histories
}
)
</script>
