'use strict';

export interface miKeywordQueryEmits {
    (e: 'request_clear_keyword_query'): void
    (e: 'request_update_use_keyword_query', use_keyword_query: boolean): void
    (e: 'request_update_keywords', keyword: string): void
    (e: 'request_update_and_search', and_search: boolean): void
}