// 編集前に読む: .claude/skills/gkill-client-tags/SKILL.md（この領域の不変条件の正本）
'use strict'

// add_tag の完了前に registered_kyou を emit してはいけない理由:
// documents/adr/0032-add-tag-before-registered-kyou.md

import type { GkillAPI } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { Tag } from '@/classes/datas/tag'
import { AddTagRequest } from '@/classes/api/req_res/add-tag-request'
import { UpdateTagRequest } from '@/classes/api/req_res/update-tag-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

/**
 * Kyouに紐づくタグの追加・論理削除。
 *
 * 同じ処理が add-tag-view / KFTL / 12本のコンテキストメニュー / 削除確認ビューへ
 * 手書きで複製されていたので、ここ1つに寄せる。
 *
 * **`tx_id` は使わない。** TXID指定時のタグは一時リポジトリにしか無いので
 * `add_tag` は `added_tag` を返せず（`handle_add_tag.go` の doc コメント）、
 * 呼び出し元は `registered_tag` を上げられなくなる。
 * しかも `commit_tx` はDBトランザクションではなく部分確定しうる
 * （`handle_commit_tx.go`）ので、束ねても原子性は買えない。
 */

/** タグ名の区切り文字。KFTLのタグ接頭辞「。」(句点)とは別物 */
export const tag_name_separator = "、"

export interface AddTagsResult {
    added_tags: Array<Tag>
    errors: Array<GkillError>
    messages: Array<GkillMessage>
}

export interface RemoveTagsResult {
    removed_tags: Array<Tag>
    errors: Array<GkillError>
    messages: Array<GkillMessage>
}

export interface ApplyTagChangesResult {
    added_tags: Array<Tag>
    removed_tags: Array<Tag>
    errors: Array<GkillError>
    messages: Array<GkillMessage>
}

/**
 * 「、」区切りの1行テキストをタグ名の配列にする。
 *
 * trim・空除去・重複除去（大小無視）まで行う。
 * サーバの重複チェックはタグIDだけを見る（`usecase/tag.go`）ので、
 * 同じ名前を2回書けば2件できてしまう。落とすのはクライアントの責任。
 */
export function parse_tag_names(text: string): Array<string> {
    const parsed = new Array<string>()
    const seen = new Set<string>()
    for (const raw of text.split(tag_name_separator)) {
        const tag_name = raw.trim()
        if (tag_name === "") {
            continue
        }
        const key = tag_name.toLowerCase()
        if (seen.has(key)) {
            continue
        }
        seen.add(key)
        parsed.push(tag_name)
    }
    return parsed
}

/**
 * タグを1件ずつ追加する。
 *
 * 呼び出し元は返った `added_tags` を `registered_tag` として emit すること
 * （上げないと ApplicationConfig のタグツリーに新しいタグが載らず、
 * サイドバーの絞り込みに出てこない）。
 *
 * 1件失敗しても残りは投げる。エラーは集約して返す
 * （途中で打ち切ると「3個書いたのに1個だけ付いた」理由が利用者に見えない）。
 */
export async function add_tags_to_target(
    gkill_api: GkillAPI,
    application_config: ApplicationConfig,
    target_id: string,
    tag_names: Array<string>,
): Promise<AddTagsResult> {
    const result: AddTagsResult = {
        added_tags: new Array<Tag>(),
        errors: new Array<GkillError>(),
        messages: new Array<GkillMessage>(),
    }
    if (tag_names.length === 0) {
        return result
    }

    for (let i = 0; i < tag_names.length; i++) {
        const now = new Date(Date.now())
        const new_tag = new Tag()
        new_tag.tag = tag_names[i]
        new_tag.id = gkill_api.generate_uuid()
        new_tag.is_deleted = false
        new_tag.target_id = target_id
        new_tag.related_time = now
        new_tag.create_app = "gkill"
        new_tag.create_device = application_config.device
        new_tag.create_time = now
        new_tag.create_user = application_config.user_id
        new_tag.update_app = "gkill"
        new_tag.update_device = application_config.device
        new_tag.update_time = now
        new_tag.update_user = application_config.user_id

        await delete_gkill_kyou_cache(new_tag.id)
        await delete_gkill_kyou_cache(new_tag.target_id)
        const req = new AddTagRequest()
        req.tag = new_tag
        const res = await gkill_api.add_tag(req)
        if (res.errors && res.errors.length !== 0) {
            result.errors.push(...res.errors)
            continue
        }
        if (res.messages && res.messages.length !== 0) {
            result.messages.push(...res.messages)
        }
        result.added_tags.push(res.added_tag)
    }

    // 履歴は実際に付いたものだけ。1つも付かなかったのに履歴が動くと
    // 次回の履歴チップが「付けられなかったタグ」で埋まる
    if (result.added_tags.length !== 0) {
        const history_value = tag_names.join(tag_name_separator)
        gkill_api.set_saved_last_added_tag(history_value)
        gkill_api.push_tag_to_history(history_value)
    }
    return result
}

/**
 * 付いているタグを論理削除する。
 *
 * gkillのリポジトリは追記のみなので、削除は `is_deleted = true` の版を足すこと
 * （`use-confirm-delete-tag-view.ts` と同じ）。
 * 呼び出し元は返った `removed_tags` を `deleted_tag` として emit すること。
 */
export async function remove_attached_tags(
    gkill_api: GkillAPI,
    application_config: ApplicationConfig,
    tags: Array<Tag>,
): Promise<RemoveTagsResult> {
    const result: RemoveTagsResult = {
        removed_tags: new Array<Tag>(),
        errors: new Array<GkillError>(),
        messages: new Array<GkillMessage>(),
    }
    if (tags.length === 0) {
        return result
    }

    for (let i = 0; i < tags.length; i++) {
        const updated_tag = tags[i].clone()
        updated_tag.is_deleted = true
        updated_tag.update_app = "gkill"
        updated_tag.update_device = application_config.device
        updated_tag.update_time = new Date(Date.now())
        updated_tag.update_user = application_config.user_id

        await delete_gkill_kyou_cache(updated_tag.id)
        await delete_gkill_kyou_cache(updated_tag.target_id)
        const req = new UpdateTagRequest()
        req.tag = updated_tag
        const res = await gkill_api.update_tag(req)
        if (res.errors && res.errors.length !== 0) {
            result.errors.push(...res.errors)
            continue
        }
        if (res.messages && res.messages.length !== 0) {
            result.messages.push(...res.messages)
        }
        result.removed_tags.push(res.updated_tag)
    }
    return result
}

/**
 * 編集画面のタグ欄の変更（足したもの・外したもの）をまとめて反映する。
 *
 * 呼び出し元は返った `added_tags` / `removed_tags` をそれぞれ
 * `registered_tag` / `deleted_tag` として emit すること。
 *
 * 追加を先にやるのは、追加が失敗したときに削除まで進めないようにするため。
 * 片方だけ反映された中途半端な状態より、元のタグが残っているほうが安全。
 */
export async function apply_kyou_tag_changes(
    gkill_api: GkillAPI,
    application_config: ApplicationConfig,
    target_id: string,
    tag_names: Array<string>,
    tags_to_remove: Array<Tag>,
): Promise<ApplyTagChangesResult> {
    const added = await add_tags_to_target(gkill_api, application_config, target_id, tag_names)
    if (added.errors.length !== 0) {
        return {
            added_tags: added.added_tags,
            removed_tags: new Array<Tag>(),
            errors: added.errors,
            messages: added.messages,
        }
    }
    const removed = await remove_attached_tags(gkill_api, application_config, tags_to_remove)
    return {
        added_tags: added.added_tags,
        removed_tags: removed.removed_tags,
        errors: removed.errors,
        messages: added.messages.concat(removed.messages),
    }
}
