<template>
    <KyouView />
</template>
<script lang="ts" setup>
import type { AddTagViewProps } from './add-tag-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { Kyou } from '@/classes/datas/kyou';
import { Tag } from '@/classes/datas/tag';
import { type Ref, ref } from 'vue';
import KyouView from './kyou-view.vue';

const props = defineProps<AddTagViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
const tag: Ref<Tag> = ref(new Tag());
</script>
