'use strict'

import { type Ref, ref } from 'vue'
import type { ManageSkillListDialogEmits } from '@/pages/dialogs/manage-skill-list-dialog-emits'
import type { ManageSkillListDialogProps } from '@/pages/dialogs/manage-skill-list-dialog-props'
import type BrowseSkillFilesDialog from '@/pages/dialogs/browse-skill-files-dialog.vue'
import type ConfirmUploadSkillDialog from '@/pages/dialogs/confirm-upload-skill-dialog.vue'
import type ConfirmDeleteSkillDialog from '@/pages/dialogs/confirm-delete-skill-dialog.vue'
import type HelpDialog from '@/pages/dialogs/help-dialog.vue'
import { GetSkillListRequest } from '@/classes/api/req_res/get-skill-list-request'
import type { SkillInfo } from '@/classes/api/req_res/get-skill-list-response'
import { DownloadSkillRequest } from '@/classes/api/req_res/download-skill-request'
import { UploadSkillRequest } from '@/classes/api/req_res/upload-skill-request'
import { DeleteSkillRequest } from '@/classes/api/req_res/delete-skill-request'
import { base64_to_blob, read_file_as_data_url } from '@/classes/file-base64'
import { save_as } from '@/classes/save-as'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

// スキル（$GKILL_HOME/skills/<user_id>/<name>/ の SKILL.md と付属ファイル。ADR-0634）の管理。
// 画面でできるのは一覧・中身の表示・zip のダウンロード・zip のアップロード（丸ごと置き換え）・
// スキル丸ごとの削除だけで、ファイル単位の編集は持たない（記録アプリの本質ではないので gkill に責務を持たせない）。
// 設定の「適用」とは独立したエンティティなので、ServerConfigDialog と同じく自分で API を呼ぶ。
export function useManageSkillListDialog(options: {
    props: ManageSkillListDialogProps
    emits: ManageSkillListDialogEmits
}) {
    const { props, emits } = options

    const browse_skill_files_dialog = ref<InstanceType<typeof BrowseSkillFilesDialog> | null>(null)
    const confirm_upload_skill_dialog = ref<InstanceType<typeof ConfirmUploadSkillDialog> | null>(null)
    const confirm_delete_skill_dialog = ref<InstanceType<typeof ConfirmDeleteSkillDialog> | null>(null)
    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)

    const skills: Ref<Array<SkillInfo>> = ref(new Array<SkillInfo>())
    const is_loading: Ref<boolean> = ref(false)
    const is_uploading: Ref<boolean> = ref(false)
    // pending_zip_base64 は確認（dry_run）を通した zip。「適用」で同じ中身を本番で送る
    const pending_zip_base64: Ref<string> = ref("")

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("manage-skill-list-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })

    async function show(): Promise<void> {
        is_show_dialog.value = true
        await reload_skills()
    }

    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        skills.value = new Array<SkillInfo>()
        pending_zip_base64.value = ""
    }

    async function reload_skills(): Promise<void> {
        is_loading.value = true
        try {
            const res = await props.gkill_api.get_skill_list(new GetSkillListRequest())
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            skills.value = res.skills ?? new Array<SkillInfo>()
        } finally {
            is_loading.value = false
        }
    }

    function show_browse_skill_files_dialog(skill: SkillInfo): void {
        browse_skill_files_dialog.value?.show(skill.name)
    }

    function show_confirm_delete_skill_dialog(skill: SkillInfo): void {
        confirm_delete_skill_dialog.value?.show(skill)
    }

    async function download_skill(skill: SkillInfo): Promise<void> {
        const req = new DownloadSkillRequest()
        req.name = skill.name
        const res = await props.gkill_api.download_skill(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        save_as(base64_to_blob(res.zip_base64, "application/zip"), res.file_name || skill.name + ".zip")
    }

    // アップロードの1段目。zip を読んで dry_run で送り、何が起きるかを確認ダイアログに出す。
    async function onSelectedUploadFile(event: Event): Promise<void> {
        const input = event.target as HTMLInputElement | null
        const file = input?.files?.[0] ?? null
        // 同じファイルを選び直しても change が起きるよう、選択を空に戻す
        if (input) {
            input.value = ""
        }
        if (file === null || is_uploading.value) {
            return
        }
        is_uploading.value = true
        try {
            const zip_base64 = await read_file_as_data_url(file)
            const req = new UploadSkillRequest()
            req.zip_base64 = zip_base64
            req.dry_run = true
            const res = await props.gkill_api.upload_skill(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.plan === null) {
                return
            }
            pending_zip_base64.value = zip_base64
            confirm_upload_skill_dialog.value?.show(res.plan)
        } finally {
            is_uploading.value = false
        }
    }

    // アップロードの2段目。確認で「適用」が押されたら、同じ zip で置き換える（DELETE_WRITE）。
    async function apply_upload_skill(): Promise<void> {
        const zip_base64 = pending_zip_base64.value
        pending_zip_base64.value = ""
        if (zip_base64 === "" || is_uploading.value) {
            return
        }
        is_uploading.value = true
        try {
            const req = new UploadSkillRequest()
            req.zip_base64 = zip_base64
            req.dry_run = false
            const res = await props.gkill_api.upload_skill(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
        } finally {
            is_uploading.value = false
        }
        await reload_skills()
    }

    async function delete_skill(skill: SkillInfo): Promise<void> {
        const req = new DeleteSkillRequest()
        req.name = skill.name
        // path を空にするとスキルを丸ごと消す（画面が使うのはこちらだけ）
        req.path = ""
        const res = await props.gkill_api.delete_skill(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        await reload_skills()
    }

    return {
        browse_skill_files_dialog,
        confirm_upload_skill_dialog,
        confirm_delete_skill_dialog,
        help_dialog,
        skills,
        is_loading,
        is_uploading,
        is_show_dialog,
        ui,
        show,
        hide,
        reload_skills,
        show_browse_skill_files_dialog,
        show_confirm_delete_skill_dialog,
        download_skill,
        onSelectedUploadFile,
        apply_upload_skill,
        delete_skill,
    }
}
