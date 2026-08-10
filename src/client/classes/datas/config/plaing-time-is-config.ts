'use strict'

import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

// plaing検索(Kyou付随の実行中表示・実行中画面・KFTLの終了候補検索)の
// カスタム検索条件。ApplicationConfig.plaing_timeis_json_data に保存される。
// null は「未設定」を表し、未設定時は従来どおり全リポジトリを対象に検索する。
export class PlaingTimeIsConfig {
    plaing_timeis_find_kyou_query: FindKyouQuery | null = null

    static parse(json: unknown): PlaingTimeIsConfig {
        const config = new PlaingTimeIsConfig()
        if (json && typeof json === 'object') {
            const obj = json as Record<string, unknown>
            if (obj.plaing_timeis_find_kyou_query) {
                config.plaing_timeis_find_kyou_query = FindKyouQuery.parse_find_kyou_query(obj.plaing_timeis_find_kyou_query)
            }
        }
        return config
    }

    to_json(): Record<string, unknown> {
        return {
            plaing_timeis_find_kyou_query: this.plaing_timeis_find_kyou_query
                ? JSON.parse(JSON.stringify(this.plaing_timeis_find_kyou_query))
                : null,
        }
    }
}
