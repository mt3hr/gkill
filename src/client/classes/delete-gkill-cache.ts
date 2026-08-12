import { CONFIG_CACHE_NAME, KYOU_CACHE_NAME } from './service-worker-utils'

export default async function delete_gkill_kyou_cache(id: string | null): Promise<void> {
    if (!id) {
        // 全消しは削除の完了まで待つ。待たずに返すと、呼び出し元が始めた引き直しが
        // 消し終わる前にキャッシュへ書き戻し、その新しい応答ごと消えてしまう
        await caches.delete(KYOU_CACHE_NAME)
        return
    }

    const data_types = [
        'kyou',
        'kmemo',
        'kc',
        'urlog',
        'nlog',
        'timeis',
        'mi',
        'lantana',
        'rekyou',
        'mirekyou',
        'git_commit_log',
        'idf_kyou',
        'tags_by_id',
        'texts_by_id',
        'gkill_notifications_by_id',
        'plugin_content_html',
    ]

    const cache = await caches.open(KYOU_CACHE_NAME)
    const wait_promises = new Array<Promise<boolean>>()
    for (let i = 0; i < data_types.length; i++) {
        const data_type = data_types[i]
        const cache_key = `/cache/api/${data_type}/${id}`
        wait_promises.push(cache.delete(new Request(cache_key)))
    }
    await Promise.all(wait_promises)
}


export async function delete_gkill_config_cache(target_data_types: Array<string> | null = null): Promise<void> {
    const data_types = target_data_types ?? [
        'application_config',
        'all_rep_names',
        'all_tag_names',
        'mi_board_list'
    ]

    const cache = await caches.open(CONFIG_CACHE_NAME)
    const wait_promises = new Array<Promise<boolean>>()
    for (let i = 0; i < data_types.length; i++) {
        const data_type = data_types[i]
        const cache_key = `/cache/api/${data_type}`
        wait_promises.push(cache.delete(new Request(cache_key)))
    }
    await Promise.all(wait_promises)
}

// タグの追加・更新でタグ名一覧が変わるため、キャッシュされた古い一覧を捨てる。
// 古い一覧を読むと編集前のタグ名がApplicationConfigのTagStructに追加されてしまう
export async function delete_gkill_all_tag_names_cache(): Promise<void> {
    try {
        await delete_gkill_config_cache(['all_tag_names'])
    } catch (_e) {
        // Cache API が利用できない環境ではスキップ
    }
}

// 付随データ(タグ/テキスト/通知)の追加・更新でそのKyouに紐づく一覧が変わるため、
// ServiceWorkerがtarget_id単位でキャッシュした /api/get_tags_by_id ・
// /api/get_texts_by_id ・ /api/get_gkill_notifications_by_id の古い一覧を捨てる。
// 捨てないと更新前の内容がKyouに付いたまま表示され続ける。
// 応答を受け取ったあとに呼ぶこと。送信前に消しても、書き込みが確定する前に
// 別のコンポーネントが引くと古い応答が入り直す
export async function delete_gkill_attached_datas_cache(target_id: string): Promise<void> {
    if (!target_id) {
        return
    }
    try {
        await delete_gkill_kyou_cache(target_id)
    } catch (_e) {
        // Cache API が利用できない環境ではスキップ
    }
}
