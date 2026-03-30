<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="use_sort_type" hide-details class="pa-0 ma-0" readonly
                    :label="i18n.global.t('SORT_TITLE')" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0 pt-2">
                <v-btn dark color="secondary" @click="emits('request_clear_sort_type')" hide-details>{{
                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-select class="select" v-model="sort_type" :items="sort_types" :label="i18n.global.t('CHECK_STATE_TITLE')"
            item-title="name" item-value="value" />
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { miSortTypeQueryEmits } from './mi-sort-type-query-emits'
import type { miSortTypeQueryProps } from './mi-sort-type-query-props'
import { useMiSortTypeQuery } from '@/classes/use-mi-sort-type-query'

const props = defineProps<miSortTypeQueryProps>()
const emits = defineEmits<miSortTypeQueryEmits>()

const {
    use_sort_type,
    sort_type,
    sort_types,
    get_sort_type,
} = useMiSortTypeQuery({ props, emits })

defineExpose({ get_sort_type })
</script>
