<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { ConfirmGenerateTLSFilesDialog } from '../dialogs/confirm-generate-tls-files-dialog.vue';
import { CreateAccountDialog } from '../dialogs/create-account-dialog.vue';
import { ManageAccountDialog } from '../dialogs/manage-account-dialog.vue';
import { ServerConfigViewEmits } from './server-config-view-emits';
import { ServerConfigViewProps } from './server-config-view-props';

// ˄

export class ServerConfigView {
    // ˅
    
    // ˄

    cloned_server_config: ServerConfig;

    is_local_only_access: Ref<boolean>;

    address: Ref<string>;

    enable_tls: Ref<boolean>;

    tls_cert_file: Ref<string>;

    tls_key_file: Ref<string>;

    open_directory_command: Ref<string>;

    open_file_command: Ref<string>;

    urlog_timeout: Ref<Number>;

    urlog_useragent: Ref<string>;

    upload_size_limit_month: Ref<Number>;

    user_data_directory: Ref<string>;

    private props: ServerConfigViewProps;

    private emits: ServerConfigViewEmits;

    private manage_account_dialog: ManageAccountDialog;

    private create_account_dialog: CreateAccountDialog;

    private confirm_generate_tls_files_dialog: ConfirmGenerateTLSFilesDialog;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>