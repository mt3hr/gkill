<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { AddTagDialog } from '../dialogs/add-tag-dialog';
import { AddTextDialog } from '../dialogs/add-text-dialog';
import { AttachedTag } from './attached-tag';
import { AttachedText } from './attached-text';
import { AttachedTimeIsPlaing } from './attached-time-is-plaing';
import { ConfirmDeleteKyouDialog } from '../dialogs/confirm-delete-kyou-dialog';
import { ConfirmDeleteTagDialog } from '../dialogs/confirm-delete-tag-dialog';
import { ConfirmDeleteTextDialog } from '../dialogs/confirm-delete-text-dialog';
import { EditIDFKyouDialog } from '../dialogs/edit-idf-kyou-dialog';
import { EditKmemoDialog } from '../dialogs/edit-kmemo-dialog';
import { EditLantanaDialog } from '../dialogs/edit-lantana-dialog';
import { EditMiDialog } from '../dialogs/edit-mi-dialog';
import { EditNlogDialog } from '../dialogs/edit-nlog-dialog';
import { EditTagDialog } from '../dialogs/edit-tag-dialog';
import { EditTextDialog } from '../dialogs/edit-text-dialog';
import { EditTimeIsDialog } from '../dialogs/edit-time-is-dialog';
import { EditURLogDialog } from '../dialogs/edit-ur-log-dialog';
import { GitCommitLogView } from './git-commit-log-view';
import { IDFKyouView } from './idf-kyou-view';
import { KmemoView } from './kmemo-view';
import { KyouHistoriesDialog } from '../dialogs/kyou-histories-dialog';
import { KyouViewEmits } from './kyou-view-emits';
import { KyouViewProps } from './kyou-view-props';
import { LantanaView } from './lantana-view';
import { MiView } from './mi-view';
import { NlogView } from './nlog-view';
import { ReKyouView } from './re-kyou-view';
import { TagHistoriesDialog } from '../dialogs/tag-histories-dialog';
import { TextHistoriesDialog } from '../dialogs/text-histories-dialog';
import { TimeIsView } from './time-is-view';
import { URLogView } from './ur-log-view.vue';

// ˄

export class KyouView {
    // ˅
    
    // ˄

    private cloned_kyou: Kyou;

    attached_tag: AttachedTag;

    attached_text: AttachedText;

    attached_timeis_plaing: AttachedTimeIsPlaing;

    kyou_dialog: KyouDialog;

    private props: KyouViewProps;

    private emits: KyouViewEmits;

    private attachedTag: Array<AttachedTag>;

    private attachedText: Array<AttachedText>;

    private attachedTimeIsPlaing: Array<AttachedTimeIsPlaing>;

    private editKmemoDialog: EditKmemoDialog;

    private editMiDialog: EditMiDialog;

    private editNlogDialog: EditNlogDialog;

    private editLantanaDialog: EditLantanaDialog;

    private editTimeIsDialog: EditTimeIsDialog;

    private editURLogDialog: EditURLogDialog;

    private editIDFKyouDialog: EditIDFKyouDialog;

    private confirmDeleteKyouDialog: ConfirmDeleteKyouDialog;

    private kyouHistoriesDialog: KyouHistoriesDialog;

    private addTagDialog: AddTagDialog;

    private editTagDialog: EditTagDialog;

    private confirmDeleteTagDialog: ConfirmDeleteTagDialog;

    private tagHistoriesDialog: TagHistoriesDialog;

    private addTextDialog: AddTextDialog;

    private editTextDialog: EditTextDialog;

    private textHistoriesDialog: TextHistoriesDialog;

    private confirmDeleteTextDialog: ConfirmDeleteTextDialog;

    private kmemoView: KmemoView;

    private miView: MiView;

    private nlogView: NlogView;

    private lantanaView: LantanaView;

    private timeIsView: TimeIsView;

    private uRLogView: URLogView;

    private iDFKyouView: IDFKyouView;

    private reKyouView: ReKyouView;

    private gitCommitLogView: GitCommitLogView;

    async load_attached_infos(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async clear_attached_infos(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async update_check(is_checked: boolean): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>