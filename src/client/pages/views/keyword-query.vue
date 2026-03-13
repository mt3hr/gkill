<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <v-checkbox v-model="cloned_find_query.use_words"
                @change="emits('request_update_use_keyword_query', cloned_find_query.use_words)"
                :label="i18n.global.t('WORD_QUERY_TITLE')" hide-details class="pa-0 ma-0" />
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pb-0 mb-0 pr-0">
            <v-btn dark color="secondary" @click="emits('request_clear_keyword_query')" hide-details>{{
                i18n.global.t("CLEAR_TITLE") }}</v-btn>
        </v-col>
    </v-row>

    <v-row v-show="cloned_find_query.use_words" class="pa-0 ma-0">
        <v-col cols="2" class="pa-0 ma-0">
            <v-btn v-if="cloned_find_query.words_and" icon="mdi-set-center"
                @click="cloned_find_query.words_and = !cloned_find_query.words_and; emits('request_update_and_search', cloned_find_query.words_and)" />
            <v-btn v-if="!cloned_find_query.words_and" icon="mdi-set-all"
                @click="cloned_find_query.words_and = !cloned_find_query.words_and; emits('request_update_and_search', cloned_find_query.words_and)" />
        </v-col>
        <v-col cols="10" class="pa-0 ma-0">
            <v-text-field v-model="cloned_find_query.keywords" :label="i18n.global.t('WORD_QUERY_TITLE')" hide-details
                @change="emits('request_update_keywords', cloned_find_query.keywords)" />
        </v-col>
    </v-row>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { KeywordQueryEmits } from './keyword-query-emits'
import type { KeywordQueryProps } from './keyword-query-props'
import { useKeywordQuery } from '@/classes/use-keyword-query'

const props = defineProps<KeywordQueryProps>()
const emits = defineEmits<KeywordQueryEmits>()

const {
    cloned_find_query,
    get_keywords,
    get_use_words,
    get_use_word_and_search,
} = useKeywordQuery({ props, emits })

defineExpose({ get_keywords, get_use_words, get_use_word_and_search })
</script>
