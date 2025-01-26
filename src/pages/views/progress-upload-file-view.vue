<template>
    <v-card>
        <v-card-title>
            ファイルアップロード中
        </v-card-title>
        <div>
            しばらくお待ち下さい
        </div>
    </v-card>
</template>
<script lang="ts" setup>
import type { ProgressUploadFileViewEmits } from './progress-upload-file-view-emits'
import type { ProgressUploadFileViewProps } from './progress-upload-file-view-props'

const props = defineProps<ProgressUploadFileViewProps>()
const emits = defineEmits<ProgressUploadFileViewEmits>()
</script>
