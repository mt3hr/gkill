'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmDeleteSkillDialogEmits } from '@/pages/dialogs/confirm-delete-skill-dialog-emits'
import type { ConfirmDeleteSkillDialogProps } from '@/pages/dialogs/confirm-delete-skill-dialog-props'
import { SkillInfo } from '@/classes/api/req_res/get-skill-list-response'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

// スキルを丸ごと消す確認。削除の API は親（use-manage-skill-list-dialog.ts）が呼ぶ。
// スキルは履歴を持たないので、消したら戻せない（ADR-0634。AI からの削除は公開していない）。
export function useConfirmDeleteSkillDialog(options: {
    props: ConfirmDeleteSkillDialogProps
    emits: ConfirmDeleteSkillDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-skill-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const skill: Ref<SkillInfo> = ref(new SkillInfo())
    async function show(target: SkillInfo): Promise<void> {
        skill.value = target
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        skill.value = new SkillInfo()
    }

    return {
        is_show_dialog,
        ui,
        skill,
        show,
        hide,
    }
}
