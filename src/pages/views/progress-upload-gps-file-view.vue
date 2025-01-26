<template>
    <v-card>
        <v-card-title>
            GPSLogファイルアップロード中
        </v-card-title>
        <div>
            しばらくお待ち下さい
        </div>
    </v-card>
</template>
<script lang="ts" setup>
import type { ProgressUploadGPSFileViewEmits } from './progress-upload-gps-file-view-emits'
import type { ProgressUploadGPSFileViewProps } from './progress-upload-gps-file-view-props'

const props = defineProps<ProgressUploadGPSFileViewProps>()
const emits = defineEmits<ProgressUploadGPSFileViewEmits>()
</script>
