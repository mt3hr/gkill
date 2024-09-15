<template>
    <ReKyouContextMenu />
</template>
<script setup lang="ts">
import type { ReKyouViewProps } from './re-kyou-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import ReKyouContextMenu from './re-kyou-context-menu.vue';
import { computed, type Ref, ref } from 'vue';
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request';
import { ReKyou } from '@/classes/datas/re-kyou';

const props = defineProps<ReKyouViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_rekyou: Ref<ReKyou> = ref(await props.rekyou.clone());
const target_kyou = computed(async () => {
    const req = new GetKyouRequest()
    req.id = props.rekyou.target_id
    const res = await props.gkill_api.get_kyou(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    return res.kyou_histories[0]
})
</script>
