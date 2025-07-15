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
        'git_commit_log',
        'idf_kyou',
        'tags_by_id',
        'texts_by_id',
        'gkill_notifications_by_id',
    ]

    const cache = await caches.open('gkill-post-kyou-cache')
    const wait_promises = new Array<Promise<any>>()
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


export async function delete_gkill_config_cache(): Promise<void> {
    const data_types = [
        'gkill_info',
        'all_rep_names',
        'all_tag_names',
        'mi_board_list'
    ]

    const cache = await caches.open('gkill-post-config-cache')
    const wait_promises = new Array<Promise<any>>()
    for (let i = 0; i < data_types.length; i++) {
        const data_type = data_types[i]
        const cacheKey = `/cache/api/${data_type}`
        wait_promises.push(cache.delete(new Request(cacheKey)))
    }
    await Promise.all(wait_promises)
}