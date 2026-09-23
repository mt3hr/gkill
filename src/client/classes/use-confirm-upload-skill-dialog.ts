'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmUploadSkillDialogEmits } from '@/pages/dialogs/confirm-upload-skill-dialog-emits'
import type { ConfirmUploadSkillDialogProps } from '@/pages/dialogs/confirm-upload-skill-dialog-props'
import { SkillReplacePlan } from '@/classes/api/req_res/upload-skill-response'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

// スキルの zip アップロードの確認（2段階の2段目）。1段目の dry_run が返した計画
// （新規か置き換えか、追加・削除・変更・無視のファイル）を見せ、「適用」で親へ依頼を返す。
// 置き換えの API は親（use-manage-skill-list-dialog.ts）が呼ぶ。zip の中身も親が持つ。
export function useConfirmUploadSkillDialog(options: {
    props: ConfirmUploadSkillDialogProps
    emits: ConfirmUploadSkillDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-upload-skill-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const plan: Ref<SkillReplacePlan> = ref(new SkillReplacePlan())
    async function show(replace_plan: SkillReplacePlan): Promise<void> {
        plan.value = replace_plan
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        plan.value = new SkillReplacePlan()
    }

    return {
        is_show_dialog,
        ui,
        plan,
        show,
        hide,
    }
}
