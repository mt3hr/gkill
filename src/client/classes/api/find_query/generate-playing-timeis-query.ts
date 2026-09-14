'use strict'

import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { PlayingTimeIsConfig } from '@/classes/datas/config/playing-time-is-config'
import { FindKyouQuery } from './find-kyou-query'

// playing検索（指定時刻に実行中のTimeIsの検索）のクエリを生成する。
// Kyou付随の実行中表示（info-base.ts の load_attached_timeis）と実行中画面
// （generate-get-playing-timeis-kyous-query.ts）がここを通る。
// ApplicationConfigに保存されたカスタム検索条件（playing_timeis_json_data）が
// あればそれを適用する。カスタム条件で候補を絞ると、条件外の実行中TimeIsは
// KFTLの/endで終了できなくなる（仕様）。
// サーバ内KFTL（kftl_timeis.go の playingTimeIsQueryFromConfig）も同じ欄を写して
// /end 系の対象を探すので、Web / Wear OS / MCP のどの経路でも同じ条件が効く（2026-09-15〜、ADR-0507）。
// 欄を増やすときは両方へ。
// GkillAPIには依存しない同期の純関数（application_configは呼び出し元が渡す）。
export function generate_playing_timeis_query(application_config: ApplicationConfig | null, playing_time: Date): FindKyouQuery {
    let query = new FindKyouQuery()
    // タグフィルタは既定で未使用（null）。旧 use_tags=false と等価
    query.tags = null
    // 共有ページ（for_share_kyou）ではrep/tagの組み立てもカスタム条件も適用しない。
    // 閲覧者の設定が共有Kyouの表示に影響・漏洩しないようにするため
    if (application_config && !application_config.for_share_kyou) {
        const saved = PlayingTimeIsConfig.parse(application_config.playing_timeis_json_data).playing_timeis_find_kyou_query
        if (saved) {
            // カスタム条件はこの明示リストの6フィールドだけをコピーする
            // （find-time-is-query-editor-view の編集面と1:1対応。
            // 片方だけ増やすと「設定したのに効かない」になるので必ず両方へ）。
            // グループの有効/無効は words/not_words/tags の null 判定が担う。
            // それ以外のフィールドは保存JSONに何が残っていても無視される
            // （記録保管場所を選べた頃の reps も含めて無視する）
            query.keywords = saved.keywords
            query.words_and = saved.words_and
            query.words = saved.words === null ? null : saved.words.concat()
            query.not_words = saved.not_words === null ? null : saved.not_words.concat()
            query.tags = saved.tags === null ? null : saved.tags.concat()
            query.tags_and = saved.tags_and
            // rep名での絞り込みはエディタから消えたので明示的に切る。
            // new FindKyouQuery() の既定は reps=[]（有効・チェック0個=0件）なので、
            // ここを放置するとサーバのrep名絞り込みで常に0件になる
            query.reps = null
            // KFTL経路は呼び出し後に parse_words_and_not_words を呼ばないため、ここで導出する
            // （冪等。キーワードグループ未使用＝words/not_words が null なら何もしない）
            query.parse_words_and_not_words()
            // 非表示タグは保存時のスナップショットではなく現在の設定から反映する
            query.apply_hide_tags(application_config)
        } else {
            // 未設定時は従来どおりの既定動作（全rep + タグフィルタ未使用）
            query = FindKyouQuery.generate_default_query_for_playing_timeis(application_config)
        }
    }
    // 基準時刻は呼び出し元の意図で常に強制する
    // （保存クエリ由来の playing_time は決して使わない。非nullの playing_time が実行中検索を表す）
    query.playing_time = playing_time
    // 記録タイプはカスタム条件の有無によらずTimeIs固定。
    // サーバのタイプ系フィルタ(find_filter.go)は和集合で、playing_time 指定が既にTimeIsのrepへ
    // 絞っているので結果は変わらない（意図を明示するための冪等な指定）
    query.rep_types = ['timeis']
    return query
}
