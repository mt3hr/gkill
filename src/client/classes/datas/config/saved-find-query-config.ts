'use strict'

import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

// 保存済み検索条件1件。id は一覧管理用のキーで、列の query_id とは無関係
// (サイドバーへの適用時は必ず列側の query_id で上書きされる)
export interface SavedFindQueryItem {
    id: string
    title: string
    find_kyou_query: FindKyouQuery
}

export class SavedFindQueryConfig {
    saved_rykv_find_kyou_querys: Array<SavedFindQueryItem> = []
    saved_mi_find_kyou_querys: Array<SavedFindQueryItem> = []

    static parse(json: unknown): SavedFindQueryConfig {
        const config = new SavedFindQueryConfig()
        if (json && typeof json === 'object') {
            const obj = json as Record<string, unknown>
            config.saved_rykv_find_kyou_querys = SavedFindQueryConfig.parse_items(obj.saved_rykv_find_kyou_querys)
            config.saved_mi_find_kyou_querys = SavedFindQueryConfig.parse_items(obj.saved_mi_find_kyou_querys)
        }
        return config
    }

    // 配列でない・find_kyou_query を持たない等の不正要素は取り込まない
    private static parse_items(json: unknown): Array<SavedFindQueryItem> {
        const items = new Array<SavedFindQueryItem>()
        if (!Array.isArray(json)) {
            return items
        }
        for (const element of json) {
            if (!element || typeof element !== 'object') {
                continue
            }
            const obj = element as Record<string, unknown>
            if (!obj.find_kyou_query || typeof obj.find_kyou_query !== 'object') {
                continue
            }
            items.push({
                id: typeof obj.id === 'string' ? obj.id : '',
                title: typeof obj.title === 'string' ? obj.title : '',
                find_kyou_query: FindKyouQuery.parse_find_kyou_query(obj.find_kyou_query),
            })
        }
        return items
    }

    // ダイアログの作業用コピーに使う(キャンセルで破棄できるよう参照を切る)
    static clone_items(items: Array<SavedFindQueryItem>): Array<SavedFindQueryItem> {
        return items.map((item) => ({
            id: item.id,
            title: item.title,
            find_kyou_query: item.find_kyou_query.clone(),
        }))
    }

    clone(): SavedFindQueryConfig {
        const config = new SavedFindQueryConfig()
        config.saved_rykv_find_kyou_querys = SavedFindQueryConfig.clone_items(this.saved_rykv_find_kyou_querys)
        config.saved_mi_find_kyou_querys = SavedFindQueryConfig.clone_items(this.saved_mi_find_kyou_querys)
        return config
    }

    to_json(): Record<string, unknown> {
        return {
            saved_rykv_find_kyou_querys: JSON.parse(JSON.stringify(this.saved_rykv_find_kyou_querys)),
            saved_mi_find_kyou_querys: JSON.parse(JSON.stringify(this.saved_mi_find_kyou_querys)),
        }
    }
}
