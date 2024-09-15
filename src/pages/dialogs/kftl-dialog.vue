<template>
    <KFTLView :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config="application_config" :gkill_api="gkill_api"
        @received_messages="(errors: Array<GkillError>) => emits('received_errors', errors)"
        @reveived_errors="(errors: Array<GkillMessage>) => emits('received_messages', errors)" />
</template>
<script lang="ts" setup>
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import type { KFTLDialogEmits } from './kftl-dialog-emits';
import type { KFTLDialogProps } from './kftl-dialog-props';
import KFTLView from '../views/kftl-view.vue';

const props = defineProps<KFTLDialogProps>();
const emits = defineEmits<KFTLDialogEmits>();
</script>
