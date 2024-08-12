// ˅
'use strict';

import { AllocateRepDialog } from '../dialogs/allocate-rep-dialog';
import { ConfirmGenerateTLSFilesDialog } from '../dialogs/confirm-generate-tls-files-dialog';
import { ConfirmResetPasswordDialog } from '../dialogs/confirm-reset-password-dialog';
import { CreateAccountDialog } from '../dialogs/create-account-dialog';
import { ManageAccountViewEmits } from './manage-account-view-emits';
import { ManageAccountViewProps } from './manage-account-view-props';
import { ShowPasswordResetLinkDialog } from '../dialogs/show-password-reset-link-dialog';

// ˄

export class ManageAccountView {
    // ˅
    
    // ˄

    cloned_server_config: ServerConfig;

    private props: ManageAccountViewProps;

    private emits: ManageAccountViewEmits;

    private allocate_rep_dialog: AllocateRepDialog;

    private create_account_dialog: CreateAccountDialog;

    private confirm_generate_tls_files_dialog: ConfirmGenerateTLSFilesDialog;

    private confirm_reset_password_dialog: ConfirmResetPasswordDialog;

    private show_password_reset_link_dialog: ShowPasswordResetLinkDialog;

    // ˅
    
    // ˄
}

// ˅

// ˄
