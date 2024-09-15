<template>

</template>
<script lang="ts" setup>
import type { GPSLogMapEmits } from './gps-log-map-emits';
import type { GPSLogMapProps } from './gps-log-map-props';

const gPSLogMapProps = defineProps<GPSLogMapProps>();
const gPSLogMapEmits = defineEmits<GPSLogMapEmits>();
</script>
