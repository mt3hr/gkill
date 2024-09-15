<template>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { KyouViewEmits } from './kyou-view-emits';
import type { TagViewProps } from './tag-view-props';
import { Tag } from '@/classes/datas/tag';

const props = defineProps<TagViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_tag: Ref<Tag> = ref(await props.tag.clone());
</script>