<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { ApplicationConfigStructContextMenu } from './application-config-struct-context-menu.vue';
import { TagStructElementEmits } from './tag-struct-element-emits';
import { TagStructElementProps } from './tag-struct-element-props';

// ˄

export class TagStructElement {
    // ˅
    
    // ˄

    private cloned_is_open: Ref<boolean>;

    private struct_list: Ref<any>;

    private is_check_when_inited: Ref<boolean>;

    private is_force_hide: Ref<boolean>;

    private props: TagStructElementProps;

    private emits: TagStructElementEmits;

    private contextmenu: ApplicationConfigStructContextMenu;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>