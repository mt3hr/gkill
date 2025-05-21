export default async function delete_gkill_cache(id: string | null): Promise<void> {
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

    if (id) {
        for (let i = 0; i < data_types.length; i++) {
            const data_type = data_types[i]
            const cacheKey = `/cache/api/${data_type}/${id}`
            const cache = await caches.open('gkill-post-cache');
            await cache.delete(new Request(cacheKey));
        }
    } else {
        caches.delete('gkill-post-cache')
    }
}