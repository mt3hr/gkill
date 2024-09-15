<template>
    <KyouView />
</template>
<script lang="ts" setup>
import type { ConfirmDeleteTagViewProps } from './confirm-delete-tag-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { Kyou } from '@/classes/datas/kyou';
import type { Tag } from '@/classes/datas/tag';
import KyouView from './kyou-view.vue';
import { type Ref, ref } from 'vue';

const props = defineProps<ConfirmDeleteTagViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_tag: Ref<Tag> = ref(await props.tag.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>