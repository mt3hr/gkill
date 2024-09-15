<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditURLogViewProps } from './edit-ur-log-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { Kyou } from '@/classes/datas/kyou';
import { URLog } from '@/classes/datas/ur-log';
import KyouView from './kyou-view.vue';

const props = defineProps<EditURLogViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_urlog: Ref<URLog> = ref(await props.urlog.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>