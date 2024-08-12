<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { ApplicationConfigStructContextMenu } from './application-config-struct-context-menu.vue';
import { RepStructElementEmits } from './rep-struct-element-emits';
import { RepStructElementProps } from './rep-struct-element-props';

// ˄

export class RepStructElement {
    // ˅
    
    // ˄

    private cloned_is_open: Ref<boolean>;

    private struct_list: Ref<any>;

    check_when_inited: Ref<boolean>;

    ignore_check_rep_rykv: Ref<boolean>;

    private props: RepStructElementProps;

    private emits: RepStructElementEmits;

    private contextmenu: ApplicationConfigStructContextMenu;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>