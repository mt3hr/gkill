<template>

</template>
<script lang="ts" setup>
import { ref, type Ref } from 'vue';
import type { SetNewPasswordViewEmits } from './set-new-password-view-emits';
import type { SetNewPasswordViewProps } from './set-new-password-view-props';


const password_reset_token: Ref<string> = ref("");
const user_id: Ref<string> = ref("");
const password: Ref<string> = ref("");
const password_sha256: Ref<string> = ref("");
const password_retype: Ref<string> = ref("");

const props = defineProps<SetNewPasswordViewProps>();
const emits = defineEmits<SetNewPasswordViewEmits>();


async function try_set_new_password(password_reset_token: string, user_id: string, password_sha_256: string): Promise<boolean> {
    throw new Error('Not implemented');
}
</script>