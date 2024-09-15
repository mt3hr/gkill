<template>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { CreateAccountViewEmits } from './create-account-view-emits';
import type { CreateAccountViewProps } from './create-account-view-props';

const props = defineProps<CreateAccountViewProps>();
const emits = defineEmits<CreateAccountViewEmits>();
const new_user_id: Ref<string> = ref("");
const do_initialize: Ref<boolean> = ref(false)
</script>
