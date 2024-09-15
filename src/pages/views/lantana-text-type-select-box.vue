<template>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import { LantanaTextTypeModel } from './lantana-text-type-model';
import type { LantanaTextTypeSelectBoxEmits } from './lantana-text-type-select-box-emits';

const emits = defineEmits<LantanaTextTypeSelectBoxEmits>();
const text_text_model: Ref<LantanaTextTypeModel> = ref(new LantanaTextTypeModel());
const kmemo_text_model: Ref<LantanaTextTypeModel> = ref(new LantanaTextTypeModel());
</script>