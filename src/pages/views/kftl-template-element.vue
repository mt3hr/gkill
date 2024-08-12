<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { ApplicationConfigStructContextMenu } from './application-config-struct-context-menu.vue';
import { KFTLTemplateElementEmits } from './kftl-template-element-emits';
import { KFTLTemplateElementProps } from './kftl-template-element-props';

// ˄

export class KFTLTemplateElement {
    // ˅
    
    // ˄

    private cloned_is_open: Ref<boolean>;

    private struct_list: Ref<any>;

    private props: KFTLTemplateElementProps;

    private emits: KFTLTemplateElementEmits;

    private contextmenu: ApplicationConfigStructContextMenu;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>