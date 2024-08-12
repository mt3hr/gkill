<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { ApplicationConfigStructContextMenu } from './application-config-struct-context-menu.vue';
import { DeviceStructElementEmits } from './device-struct-element-emits';
import { DeviceStructElementProps } from './device-struct-element-props';

// ˄

export class DeviceStructElement {
    // ˅
    
    // ˄

    private cloned_is_open: Ref<boolean>;

    private struct_list: Ref<any>;

    check_when_inited: Ref<boolean>;

    private props: DeviceStructElementProps;

    private emits: DeviceStructElementEmits;

    private contextmenu: ApplicationConfigStructContextMenu;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>