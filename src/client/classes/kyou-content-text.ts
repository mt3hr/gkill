'use strict'

import { i18n } from '@/i18n'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { GkillError } from '@/classes/api/gkill-error'
import { GetPluginContentHTMLRequest } from '@/classes/api/req_res/get-plugin-content-html-request'
import type { GkillAPI } from '@/classes/api/gkill-api'
import type { Kyou } from '@/classes/datas/kyou'

// ReKyouの参照先をたどる深さの上限。循環参照で無限ループしないようにする
const MAX_REKYOU_DEPTH = 8

/**
 * KyouのHTMLからプレーンテキストを取り出す。
 * プラグインのContent HTMLは見た目のためのCSS/JSを多く含むので、それらを落としてから本文だけを取る。
 */
export function extract_text_from_html(html: string): string {
    const document_from_html = new DOMParser().parseFromString(html, 'text/html')
    document_from_html.querySelectorAll('script, style, noscript, template').forEach((element) => element.remove())
    const text = document_from_html.body?.textContent ?? ''
    return normalize_text(text)
}

function normalize_text(text: string): string {
    return text
        .replace(/\r\n?/g, '\n')
        .split('\n')
        .map((line) => line.trim())
        .join('\n')
        .replace(/\n{3,}/g, '\n\n')
        .trim()
}

function join_content_parts(parts: Array<string | number | null | undefined>): string {
    return parts
        .filter((part) => part !== null && part !== undefined && String(part).length !== 0)
        .map((part) => String(part))
        .join(' ')
}

/**
 * Kyouの本文をテキストとして取得する。
 * 日時やタグは含めず、種別ごとの本文にあたるフィールドのみを返す。
 * どの種別にも該当しない場合は空文字を返す。
 */
export async function get_kyou_content_text(kyou: Kyou, gkill_api: GkillAPI, depth: number = 0): Promise<{ text: string, errors: Array<GkillError> }> {
    const no_errors = new Array<GkillError>()

    if (kyou.typed_kmemo) {
        return { text: normalize_text(kyou.typed_kmemo.content), errors: no_errors }
    }
    if (kyou.typed_kc) {
        return { text: join_content_parts([kyou.typed_kc.title, kyou.typed_kc.num_value]), errors: no_errors }
    }
    if (kyou.typed_urlog) {
        return { text: kyou.typed_urlog.url, errors: no_errors }
    }
    if (kyou.typed_nlog) {
        return { text: join_content_parts([kyou.typed_nlog.shop, kyou.typed_nlog.title, kyou.typed_nlog.amount]), errors: no_errors }
    }
    if (kyou.typed_timeis) {
        return { text: kyou.typed_timeis.title, errors: no_errors }
    }
    if (kyou.typed_mi) {
        return { text: kyou.typed_mi.title, errors: no_errors }
    }
    if (kyou.typed_lantana) {
        return { text: String(kyou.typed_lantana.mood), errors: no_errors }
    }
    if (kyou.typed_idf_kyou) {
        return { text: kyou.typed_idf_kyou.file_name, errors: no_errors }
    }
    if (kyou.typed_git_commit_log) {
        return { text: normalize_text(kyou.typed_git_commit_log.commit_message), errors: no_errors }
    }
    if (kyou.typed_rekyou) {
        const attached_kyou = kyou.typed_rekyou.attached_kyou
        if (!attached_kyou || depth >= MAX_REKYOU_DEPTH) {
            return { text: '', errors: no_errors }
        }
        return get_kyou_content_text(attached_kyou, gkill_api, depth + 1)
    }
    if (kyou.typed_plugin) {
        const req = new GetPluginContentHTMLRequest()
        req.rep_name = kyou.typed_plugin.rep_name
        req.kyou_id = kyou.id
        const res = await gkill_api.get_plugin_content_html(req)
        if (res.errors && res.errors.length !== 0) {
            return { text: '', errors: res.errors }
        }
        return { text: extract_text_from_html(res.html), errors: no_errors }
    }

    return { text: '', errors: no_errors }
}

/**
 * Kyouの本文をクリップボードにコピーする。
 * 本文が取得できなかった場合はクリップボードを書き換えず、メッセージも返さない。
 */
export async function copy_kyou_content(kyou: Kyou, gkill_api: GkillAPI): Promise<{ messages: Array<GkillMessage>, errors: Array<GkillError> }> {
    const messages = new Array<GkillMessage>()
    const { text, errors } = await get_kyou_content_text(kyou, gkill_api)
    if (errors.length !== 0) {
        return { messages: messages, errors: errors }
    }
    if (text.length === 0) {
        return { messages: messages, errors: errors }
    }

    await navigator.clipboard.writeText(text)

    const message = new GkillMessage()
    message.message_code = GkillMessageCodes.copied_kyou_content
    message.message = i18n.global.t("COPIED_CONTENT_MESSAGE")
    messages.push(message)
    return { messages: messages, errors: errors }
}
