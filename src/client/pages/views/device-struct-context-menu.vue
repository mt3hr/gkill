<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="emits('requested_edit_device', id)">
                <v-list-item-title>{{ i18n.global.t("EDIT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_delete_device', id)">
                <v-list-item-title>{{ i18n.global.t("DELETE_TITLE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { DeviceStructContextMenuEmits } from './device-struct-context-menu-emits';
import type { DeviceStructContextMenuProps } from './device-struct-context-menu-props';
import { useDeviceStructContextMenu } from '@/classes/use-device-struct-context-menu';

const props = defineProps<DeviceStructContextMenuProps>()
const emits = defineEmits<DeviceStructContextMenuEmits>()

const {
    id,
    is_show,
    context_menu_style,
    show,
    hide,
} = useDeviceStructContextMenu({ props, emits })

defineExpose({ show, hide })
</script>
