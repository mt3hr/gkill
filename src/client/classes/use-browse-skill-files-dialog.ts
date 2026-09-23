'use strict'

import { type Ref, ref } from 'vue'
import type { BrowseSkillFilesDialogEmits } from '@/pages/dialogs/browse-skill-files-dialog-emits'
import type { BrowseSkillFilesDialogProps } from '@/pages/dialogs/browse-skill-files-dialog-props'
import { GetSkillRequest } from '@/classes/api/req_res/get-skill-request'
import type { SkillDetail, SkillFileInfo } from '@/classes/api/req_res/get-skill-response'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

// SKILL_MANIFEST_PATH はスキルの本体。一覧を開いたときはこれを表示する（中身は一覧の応答に入っている）。
const SKILL_MANIFEST_PATH = "SKILL.md"

// スキルの中身を見るだけのダイアログ（編集はしない。直すなら zip をダウンロードして上げ直す）。
// テキストは素の文字として <pre> に出す。Markdown や HTML として描かない —— AI が書いた
// HTML の中のスクリプトがログイン中のセッションで動く穴になるため（ADR-0634）。
export function useBrowseSkillFilesDialog(options: {
    props: BrowseSkillFilesDialogProps
    emits: BrowseSkillFilesDialogEmits
}) {
    const { props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("browse-skill-files-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })

    const skill: Ref<SkillDetail | null> = ref(null)
    const selected_path: Ref<string> = ref(SKILL_MANIFEST_PATH)
    const text: Ref<string> = ref("")
    const is_binary: Ref<boolean> = ref(false)
    const is_loading: Ref<boolean> = ref(false)
    // 応答の追い越し対策。ファイルを続けて押すと、先に投げた要求の応答が後から来て表示を上書きしうる
    let load_seq = 0

    async function show(name: string): Promise<void> {
        is_show_dialog.value = true
        await load_skill(name)
    }

    async function hide(): Promise<void> {
        load_seq++
        close_dialog_via_history(is_show_dialog)
        skill.value = null
        selected_path.value = SKILL_MANIFEST_PATH
        text.value = ""
        is_binary.value = false
        is_loading.value = false
    }

    async function load_skill(name: string): Promise<void> {
        const seq = ++load_seq
        is_loading.value = true
        try {
            const req = new GetSkillRequest()
            req.name = name
            const res = await props.gkill_api.get_skill(req)
            if (seq !== load_seq) {
                return
            }
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            skill.value = res.skill
            selected_path.value = SKILL_MANIFEST_PATH
            text.value = res.skill?.content ?? ""
            is_binary.value = false
        } finally {
            if (seq === load_seq) {
                is_loading.value = false
            }
        }
    }

    async function select_file(file: SkillFileInfo): Promise<void> {
        const current = skill.value
        if (current === null) {
            return
        }
        const seq = ++load_seq
        selected_path.value = file.path
        if (!file.is_text) {
            text.value = ""
            is_binary.value = true
            is_loading.value = false
            return
        }
        is_binary.value = false
        if (file.path === SKILL_MANIFEST_PATH) {
            text.value = current.content
            is_loading.value = false
            return
        }
        is_loading.value = true
        try {
            const req = new GetSkillRequest()
            req.name = current.name
            req.path = file.path
            const res = await props.gkill_api.get_skill(req)
            if (seq !== load_seq) {
                return
            }
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            text.value = res.file?.content ?? ""
            is_binary.value = res.file !== null && !res.file.is_text
        } finally {
            if (seq === load_seq) {
                is_loading.value = false
            }
        }
    }

    return {
        is_show_dialog,
        ui,
        skill,
        selected_path,
        text,
        is_binary,
        is_loading,
        show,
        hide,
        select_file,
    }
}
