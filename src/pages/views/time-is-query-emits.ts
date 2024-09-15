'use strict';

export interface TimeIsQueryEmits {
    (e: 'request_clear_timeis_query'): void
    (e: 'request_update_use_timeis_query', use_timeis_query: boolean): void
    (e: 'request_update_timeis_keywords', timeis_keyword: string): void
    (e: 'request_update_and_search_timeis_word', and_search_timeis_word: boolean): void
    (e: 'request_update_and_search_timeis_tags', and_search_timeis_tags: boolean): void
}