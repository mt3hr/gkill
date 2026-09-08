'use strict'

import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

// playing検索(Kyou付随の実行中表示・実行中画面・KFTLの終了候補検索)の
// カスタム検索条件。ApplicationConfig.playing_timeis_json_data に保存される。
// null は「未設定」を表し、未設定時は従来どおり全リポジトリを対象に検索する。
export class PlayingTimeIsConfig {
    playing_timeis_find_kyou_query: FindKyouQuery | null = null

    static parse(json: unknown): PlayingTimeIsConfig {
        const config = new PlayingTimeIsConfig()
        if (json && typeof json === 'object') {
            const obj = json as Record<string, unknown>
            if (obj.playing_timeis_find_kyou_query) {
                config.playing_timeis_find_kyou_query = FindKyouQuery.parse_find_kyou_query(obj.playing_timeis_find_kyou_query)
            }
        }
        return config
    }

    to_json(): Record<string, unknown> {
        return {
            playing_timeis_find_kyou_query: this.playing_timeis_find_kyou_query
                ? JSON.parse(JSON.stringify(this.playing_timeis_find_kyou_query))
                : null,
        }
    }
}
