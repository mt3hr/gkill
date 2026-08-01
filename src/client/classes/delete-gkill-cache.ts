export default async function delete_gkill_kyou_cache(id: string | null): Promise<void> {
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

    const cache = await caches.open('gkill-post-kyou-cache')
    const wait_promises = new Array<Promise<boolean>>()
    if (id) {
        for (let i = 0; i < data_types.length; i++) {
            const data_type = data_types[i]
            const cacheKey = `/cache/api/${data_type}/${id}`
            wait_promises.push(cache.delete(new Request(cacheKey)))
        }
    } else {
        caches.delete('gkill-post-kyou-cache')
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

    const cache = await caches.open('gkill-post-config-cache')
    const wait_promises = new Array<Promise<boolean>>()
    for (let i = 0; i < data_types.length; i++) {
        const data_type = data_types[i]
        const cacheKey = `/cache/api/${data_type}`
        wait_promises.push(cache.delete(new Request(cacheKey)))
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

// タグの追加・更新でそのKyouに紐づくタグ一覧が変わるため、ServiceWorkerが
// target_id単位でキャッシュした /api/get_tags_by_id の古い一覧を捨てる。
// 捨てないと改名前のタグ名がKyouに付いたまま表示され続ける
export async function delete_gkill_attached_tags_cache(target_id: string): Promise<void> {
    if (!target_id) {
        return
    }
    try {
        await delete_gkill_kyou_cache(target_id)
    } catch (_e) {
        // Cache API が利用できない環境ではスキップ
    }
}