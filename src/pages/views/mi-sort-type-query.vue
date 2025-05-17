<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="use_sort_type" hide-details class="pa-0 ma-0" readonly :label="i18n.global.t('SORT_TITLE')" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0">
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
import { nextTick, type Ref, ref, watch } from 'vue'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import type { miSortTypeQueryEmits } from './mi-sort-type-query-emits'
import type { miSortTypeQueryProps } from './mi-sort-type-query-props'

const props = defineProps<miSortTypeQueryProps>()
const emits = defineEmits<miSortTypeQueryEmits>()

const query = ref(props.find_kyou_query.clone())
defineExpose({ get_sort_type })

watch(() => props.find_kyou_query, () => {
    query.value = props.find_kyou_query.clone()
    load_sort_type()
})

nextTick(() => {
    load_sort_type()
    emits('inited')
})

const use_sort_type = ref(true)
const sort_type: Ref<MiSortType> = ref(MiSortType.create_time)

watch(() => sort_type.value, () => {
    emits('request_update_sort_type', sort_type.value)
})

function load_sort_type(): void {
    for (let i = 0; i < sort_types.value.length; i++) {
        if (sort_types.value[i].value === query.value.mi_sort_type) {
            sort_type.value = sort_types.value[i].value
            break
        }
    }
}

const sort_types: Ref<Array<{ name: string, value: MiSortType }>> = ref([
    {
        name: i18n.global.t("MI_CREATE_DATE_TIME_TITLE"),
        value: MiSortType.create_time,
    },
    {
        name: i18n.global.t("MI_START_DATE_TIME_TITLE"),
        value: MiSortType.estimate_start_time,
    },
    {
        name: i18n.global.t("MI_END_DATE_TIME_TITLE"),
        value: MiSortType.estimate_end_time,
    },
    {
        name: i18n.global.t("MI_LIMIT_DATE_TIME_TITLE"),
        value: MiSortType.limit_time,
    }
])

function get_sort_type(): MiSortType {
    return sort_type.value
}
</script>
