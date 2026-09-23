'use strict'

import type { SkillInfo } from "@/classes/api/req_res/get-skill-list-response"

export interface ManageSkillListViewEmits {
    (e: 'requested_browse_skill', skill: SkillInfo): void
    (e: 'requested_download_skill', skill: SkillInfo): void
    (e: 'requested_show_confirm_delete_skill_dialog', skill: SkillInfo): void
    // selected_upload_file はファイル選択の change イベントをそのまま渡す（読み取りと入力欄のリセットは親がする）
    (e: 'selected_upload_file', event: Event): void
}
