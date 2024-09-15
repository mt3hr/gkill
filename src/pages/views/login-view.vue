<template>
    <!-- //TODO -->
    <div></div>
</template>
<script lang="ts" setup>
import { ref, type Ref } from 'vue';
import { type LoginViewProps } from './login-view-props';
import type LoginViewEmits from './login-view-emits';

const user_id: Ref<string> = ref("");
const password: Ref<string> = ref("");
const password_sha256: Ref<string> = ref("");

const loginViewProps = defineProps<LoginViewProps>();
const loginViewEmits = defineEmits<LoginViewEmits>();

async function try_login(user_id: string, password_sha256: string): Promise<boolean> {
    throw new Error('Not implemented');
}

</script>