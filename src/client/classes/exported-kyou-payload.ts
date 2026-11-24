'use strict'

// 「HTML で出力」したファイルへ焼き込まれた JSON を、通信をしない GkillAPI へ流し込む。
//
// 焼き込みは `<script type="application/json" id="gkill-export-data">` に入っていて、
// サーバ（export_html_render.go）が1行を丸ごと置き換えて作る。
// JSON.parse の結果はただのオブジェクトでメソッドが生えていないので、
// hydrate_all でデータクラスへ詰め替える（日時のキーは Date に戻る）。
//
// **サーバへ取りに行くフォールバックを足してはいけない。** 読めないものは落として、
// 画面側は「記録が含まれていません」を出して終わる。

import { GkillAPIForExportedKyou } from '@/classes/api/gkill-api'
import { hydrate, hydrate_all } from '@/classes/api/hydrate'
import { base64_to_bytes } from '@/classes/file-base64'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { does_kyou_match_query } from '@/classes/kyou-local-insert'
import { deep_equals } from '@/classes/deep-equals'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { Kyou } from '@/classes/datas/kyou'
import { Kmemo } from '@/classes/datas/kmemo'
import { KC } from '@/classes/datas/kc'
import { TimeIs } from '@/classes/datas/time-is'
import { Mi } from '@/classes/datas/mi'
import { Nlog } from '@/classes/datas/nlog'
import { Lantana } from '@/classes/datas/lantana'
import { URLog } from '@/classes/datas/ur-log'
import { IDFKyou } from '@/classes/datas/idf-kyou'
import { ReKyou } from '@/classes/datas/re-kyou'
import { MiReKyou } from '@/classes/datas/mi-re-kyou'
import { GitCommitLog } from '@/classes/datas/git-commit-log'
import { GPSLog } from '@/classes/datas/gps-log'
import { Tag } from '@/classes/datas/tag'
import { Text } from '@/classes/datas/text'
import { Notification } from '@/classes/datas/notification'

/** 出力できる画面。焼き込みの印と、出力の要求に載せる値 */
export type ExportedScreen = 'rykv' | 'mi'

/** 既定の画面。印が無い古い出力はこれとして読む */
export const exported_screen_default: ExportedScreen = 'rykv'

function as_screen(value: unknown): ExportedScreen {
    return value === 'mi' ? 'mi' : exported_screen_default
}

/** 焼き込み JSON の形。欠けているキーは空として扱う（古い出力でも落ちない） */
export interface ExportedKyouPayload {
    schema_version: number
    /**
     * どの画面として出力したか。**出力ページがマウント先を選ぶのはこれ1つだけ**で、
     * ビューの中へモードのフラグを戻さない（ADR-0418 / ADR-0420）。
     *
     * 知らない値と欠損は `rykv` へ寄せる。印を足す前の出力はこのキーを持たないので、
     * **これだけのために schema_version を上げる必要は無い**。
     */
    screen: ExportedScreen
    title: string
    locale_name: string
    with_dnote: boolean
    with_map: boolean
    with_sidebar: boolean
    warnings: Array<string>
}

/** いま読める焼き込みの世代。サーバ側（export_html_render.go）と対 */
export const exported_payload_schema_version = 1

function as_record(value: unknown): Record<string, unknown> {
    if (!value || typeof value !== 'object') {
        return {}
    }
    return value as Record<string, unknown>
}

function as_array(value: unknown): Array<unknown> {
    return Array.isArray(value) ? value : []
}

function as_string(value: unknown, fallback: string): string {
    return typeof value === 'string' ? value : fallback
}

/**
 * 焼き込み JSON から「出力の説明」だけを読む。
 *
 * `schema_version` が数値でなければ null を返す。ガワだけの template.html を直接開くと
 * 中身が `{}` なのでここに落ちる。
 */
export function parse_exported_payload(raw: unknown): ExportedKyouPayload | null {
    if (!raw || typeof raw !== 'object') {
        return null
    }
    const source = raw as Record<string, unknown>
    if (typeof source.schema_version !== 'number') {
        return null
    }
    return {
        schema_version: source.schema_version,
        screen: as_screen(source.screen),
        title: as_string(source.title, ''),
        locale_name: as_string(source.locale_name, 'ja'),
        with_dnote: source.with_dnote === true,
        with_map: source.with_map === true,
        with_sidebar: source.with_sidebar === true,
        warnings: as_array(source.warnings).filter((w): w is string => typeof w === 'string'),
    }
}

/**
 * 焼き込み JSON を GkillAPIForExportedKyou へ流し込み、そのインスタンスを返す。
 *
 * 列は `kyou_ids` の参照で持つ（列をまたいで同じ記録が出るのは普通で、
 * 列ごとに本体を持つと素直に膨らむ）。ここで ID から本体へ解決する。
 */
export function build_exported_api(raw: unknown): GkillAPIForExportedKyou {
    const source = as_record(raw)
    const api = GkillAPIForExportedKyou.get_instance_for_exported_kyou()

    api.kyous = hydrate_all(source.kyous, () => new Kyou())
    api.kmemos = hydrate_all(source.kmemos, () => new Kmemo())
    api.kcs = hydrate_all(source.kcs, () => new KC())
    api.timeiss = hydrate_all(source.timeiss, () => new TimeIs())
    api.mis = hydrate_all(source.mis, () => new Mi())
    api.nlogs = hydrate_all(source.nlogs, () => new Nlog())
    api.lantanas = hydrate_all(source.lantanas, () => new Lantana())
    api.urlogs = hydrate_all(source.urlogs, () => new URLog())
    api.idf_kyous = hydrate_all(source.idf_kyous, () => new IDFKyou())
    api.rekyous = hydrate_all(source.rekyous, () => new ReKyou())
    api.mirekyous = hydrate_all(source.mirekyous, () => new MiReKyou())
    api.git_commit_logs = hydrate_all(source.git_commit_logs, () => new GitCommitLog())
    api.gps_logs = hydrate_all(source.gps_logs, () => new GPSLog())
    api.attached_tags = hydrate_all(source.attached_tags, () => new Tag())
    api.attached_texts = hydrate_all(source.attached_texts, () => new Text())
    api.attached_notifications = hydrate_all(source.attached_notifications, () => new Notification())
    api.attached_timeiss = hydrate_all(source.attached_timeiss, () => new TimeIs())
    api.attached_timeis_kyous = hydrate_all(source.attached_timeis_kyous, () => new Kyou())

    // 設定（集計 Dnote の定義 dnote_json_data もここ）。
    // google_map_api_key は焼き込まれていない（サーバが空へ潰す）
    api.exported_application_config = hydrate(new ApplicationConfig(), source.application_config)
    // 焼き込んだ設定は定義上「読み込み済み」。rykv の init() は
    // application_config.is_loaded の watch で起動するので、false のままだと
    // 列が1本も作られず**読み込み中オーバーレイが消えない**（焼き込み側の値には頼らない）
    api.exported_application_config.is_loaded = true
    api.exported_application_config.for_export_kyou = true
    // 出力では列の「検索」は焼いた配列を読むだけなので、必ず初期検索を走らせる。
    // 利用者の設定が false のまま焼かれると、init() が検索を1本も投げず
    // **列の枠だけが出て中身が空のまま**になる（検索ボタンを押すまで気づけない）
    api.exported_application_config.rykv_hot_reload = true

    // プラグインの本文。鍵は rep 名と Kyou の ID の対
    api.plugin_html = new Map()
    const plugin_html = as_record(source.plugin_html)
    for (const key in plugin_html) {
        const html = plugin_html[key]
        if (typeof html === 'string') {
            api.plugin_html.set(key, html)
        }
    }

    // メディア。鍵は build_media_url が組み立てた URL（クエリ込み）
    api.media = new Map()
    api.dropped_media = new Set()
    const media = as_record(source.media)
    for (const key in media) {
        const entry = as_record(media[key])
        // 予算に収まらなかったもの。中身は無いが、鍵は残っている
        if (entry.kind === 'dropped') {
            api.dropped_media.add(key)
            continue
        }
        if (typeof entry.src === 'string' && entry.src !== '') {
            api.media.set(key, entry.src)
        }
    }

    // IDF のテキスト本文。**バイト列のまま**持つ（文字コードの判定は
    // detect_and_decode_text が唯一の判定者で、サーバ側に書くと2実装になって必ずずれる）
    api.idf_text_bytes = new Map()
    const idf_text_contents = as_record(source.idf_text_contents)
    for (const key in idf_text_contents) {
        const entry = as_record(idf_text_contents[key])
        if (typeof entry.bytes_base64 !== 'string') {
            continue
        }
        try {
            api.idf_text_bytes.set(key, base64_to_bytes(entry.bytes_base64))
        } catch {
            // 壊れた base64 は飛ばす。その記録の本文が出ないだけで、画面は出す
        }
    }

    // タグは焼き込みから**先に埋める**。遅延ロード（load_attached_tags）待ちのまま
    // 絞り込むと、「まだ読んでいないので持っていない」と判定してタグ条件が全部落ちる
    const tags_by_target = new Map<string, Array<Tag>>()
    for (const tag of api.attached_tags) {
        const list = tags_by_target.get(tag.target_id)
        if (list) {
            list.push(tag)
        } else {
            tags_by_target.set(tag.target_id, [tag])
        }
    }
    for (const kyou of api.kyous) {
        kyou.attached_tags = tags_by_target.get(kyou.id) ?? []
        kyou.is_attached_tags_loaded = true
    }

    // 出力先サイドバーの絞り込み。
    //
    // **焼いた配列を起点に減らすことしかしない。** どんな条件を入れても、
    // 焼かれていない記録が現れることはない（これが共有の境界線）。
    //
    // 判定は kyou-local-insert.ts の does_kyou_match_query をそのまま使う。
    // これは find_filter.go の意味論の写しとして既にあるもので、
    // 出力のためにもう1つ書くと3実装目になってドリフトする。
    // ワード・TimeIs・地図・playing は見ない（Kyou 単体では判定できない条件。
    // can_decide_query_locally が同じ線引きを持っている）ので、
    // それらを理由に記録が落ちることはない。
    api.narrow_column_kyous = (kyous, query, baked_query) => {
        // 出力した時点の条件そのままなら、焼いた配列をそのまま返す。
        // 絞り込まないと、サーバが本文一致で入れた記録を
        // 「この条件では一致しない」と判定して**出力直後の画面が欠ける**
        if (deep_equals(query, baked_query)) {
            return kyous
        }
        return kyous.filter(kyou => does_kyou_match_query(kyou, query))
    }

    // 列。kyou_ids から本体へ解決する
    const kyou_by_id = new Map<string, Kyou>()
    for (let i = 0; i < api.kyous.length; i++) {
        kyou_by_id.set(api.kyous[i].id, api.kyous[i])
    }
    api.columns = as_array(source.columns).map((raw_column) => {
        const column = as_record(raw_column)
        const query = FindKyouQuery.parse_find_kyou_query(column.query ?? {})
        const kyous: Array<Kyou> = []
        for (const id of as_array(column.kyou_ids)) {
            if (typeof id !== 'string') {
                continue
            }
            const kyou = kyou_by_id.get(id)
            if (kyou) {
                kyous.push(kyou)
            }
        }
        return { query, kyous }
    })

    return api
}
