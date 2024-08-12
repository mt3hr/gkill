<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { AddRepDialog } from '../dialogs/add-rep-dialog';
import { AllocateRepViewEmits } from './allocate-rep-view-emits';
import { AllocateRepViewProps } from './allocate-rep-view-props';
import { ConfirmDeleteRepDialog } from '../dialogs/confirm-delete-rep-dialog.vue';

// ˄

export class AllocateRepView {
    // ˅
    
    // ˄

    cloned_server_config: ServerConfig;

    private props: AllocateRepViewProps;

    private emits: AllocateRepViewEmits;

    private confirm_delete_rep_dialog: ConfirmDeleteRepDialog;

    private add_rep_dialog: AddRepDialog;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>