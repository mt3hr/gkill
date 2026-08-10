'use strict'

import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { PlaingTimeIsConfig } from '@/classes/datas/config/plaing-time-is-config'
import { FindKyouQuery } from './find-kyou-query'

// plaing検索（指定時刻に実行中のTimeIsの検索）のクエリを生成する。
// Kyou付随の実行中表示（info-base.ts の load_attached_timeis）・実行中画面・
// KFTLの/end系終了候補検索（generate-get-plaing-timeis-kyous-query.ts）の
// すべてがここを通る。
// ApplicationConfigに保存されたカスタム検索条件（plaing_timeis_json_data）が
// あればそれを適用する。カスタム条件で候補を絞ると、条件外の実行中TimeIsは
// KFTLの/endで終了できなくなる（仕様）。
// Wear OS（GkillApiClient.kt）とサーバ内KFTL（kftl_timeis.go）のplaing検索は
// 別系統のため、この設定は効かない。
// GkillAPIには依存しない同期の純関数（application_configは呼び出し元が渡す）。
export function generate_plaing_timeis_query(application_config: ApplicationConfig | null, plaing_time: Date): FindKyouQuery {
    let query = new FindKyouQuery()
    // タグフィルタは既定で未使用（null）。旧 use_tags=false と等価
    query.tags = null
    // 共有ページ（for_share_kyou）ではrep/tagの組み立てもカスタム条件も適用しない。
    // 閲覧者の設定が共有Kyouの表示に影響・漏洩しないようにするため
    if (application_config && !application_config.for_share_kyou) {
        const saved = PlaingTimeIsConfig.parse(application_config.plaing_timeis_json_data).plaing_timeis_find_kyou_query
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
            query = FindKyouQuery.generate_default_query_for_plaing_timeis(application_config)
        }
    }
    // 基準時刻は呼び出し元の意図で常に強制する
    // （保存クエリ由来の plaing_time は決して使わない。非nullの plaing_time が実行中検索を表す）
    query.plaing_time = plaing_time
    // 記録タイプはカスタム条件の有無によらずTimeIs固定。
    // サーバのタイプ系フィルタ(find_filter.go)は和集合で、plaing_time 指定が既にTimeIsのrepへ
    // 絞っているので結果は変わらない（意図を明示するための冪等な指定）
    query.rep_types = ['timeis']
    return query
}
