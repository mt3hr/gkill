'use strict'

import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

export class DashboardConfig {
    dashboard_mi_find_kyou_query: FindKyouQuery | null = null
    dashboard_dnote_find_kyou_query: FindKyouQuery | null = null

    static parse(json: unknown): DashboardConfig {
        const config = new DashboardConfig()
        if (json && typeof json === 'object') {
            const obj = json as Record<string, unknown>
            if (obj.dashboard_mi_find_kyou_query) {
                config.dashboard_mi_find_kyou_query = FindKyouQuery.parse_find_kyou_query(obj.dashboard_mi_find_kyou_query)
            }
            if (obj.dashboard_dnote_find_kyou_query) {
                config.dashboard_dnote_find_kyou_query = FindKyouQuery.parse_find_kyou_query(obj.dashboard_dnote_find_kyou_query)
            }
            // 後方互換: 旧フィールド名からマイグレーション
            if (!config.dashboard_mi_find_kyou_query && obj.dashboard_default_find_kyou_query) {
                config.dashboard_mi_find_kyou_query = FindKyouQuery.parse_find_kyou_query(obj.dashboard_default_find_kyou_query)
            }
        }
        return config
    }

    to_json(): Record<string, unknown> {
        return {
            dashboard_mi_find_kyou_query: this.dashboard_mi_find_kyou_query
                ? JSON.parse(JSON.stringify(this.dashboard_mi_find_kyou_query))
                : null,
            dashboard_dnote_find_kyou_query: this.dashboard_dnote_find_kyou_query
                ? JSON.parse(JSON.stringify(this.dashboard_dnote_find_kyou_query))
                : null,
        }
    }
}
