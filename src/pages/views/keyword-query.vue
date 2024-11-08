<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <v-checkbox v-model="use_words" @click="emits('request_update_use_keyword_query', use_words)" label="キーワード"
                hide-details class="pa-0 ma-0" />
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pb-0 mb-0 pr-0">
            <v-btn @click="emits('request_clear_keyword_query')" hide-details>クリア</v-btn>
        </v-col>
    </v-row>

    <v-row v-show="use_words" class="pa-0 ma-0">
        <v-col cols="2" class="pa-0 ma-0">
            <v-btn v-if="!use_and_search" icon="mdi-set-center" @click="use_and_search = !use_and_search" />
            <v-btn v-if="use_and_search" icon="mdi-set-all" @click="use_and_search = !use_and_search" />
        </v-col>
        <v-col cols="10" class="pa-0 ma-0">
            <v-text-field v-model="keywords" label="キーワード" hide-details />
        </v-col>
    </v-row>
</template>
<script lang="ts" setup>
import type { KeywordQueryEmits } from './keyword-query-emits'
import type { KeywordQueryProps } from './keyword-query-props'
import { ref, type Ref } from 'vue'

const props = defineProps<KeywordQueryProps>()
const emits = defineEmits<KeywordQueryEmits>()

const keywords: Ref<string> = ref("")
const use_words: Ref<boolean> = ref(true)
const use_and_search: Ref<boolean> = ref(true)
</script>
