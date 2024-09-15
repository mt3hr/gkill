'use strict';

export interface miTagQueryEmits {
    (e: 'request_clear_tag_query'): void
    (e: 'request_update_checked_tags', checked_tags: Array<string>): void
    (e: 'request_update_and_search_tags', and_search_tags: boolean): void
}