<template>
    <v-card class="pa-2">
        <v-card-title>
            タグ構造削除
        </v-card-title>
        {{ device_struct.device_name }}
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="emits('requested_delete_device', device_struct.id)">削除</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import type { ConfirmDeleteDeviceStructViewProps } from './confirm-delete-device-struct-view-props.ts';
import type { ConfirmDeleteDeviceStructViewEmits } from './confirm-delete-device-struct-view-emits.ts';

defineProps<ConfirmDeleteDeviceStructViewProps>()
const emits = defineEmits<ConfirmDeleteDeviceStructViewEmits>()
</script>
