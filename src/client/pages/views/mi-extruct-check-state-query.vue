<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="use_mi_check_state" readonly hide-details class="pa-0 ma-0"
                    :label="i18n.global.t('CHECK_STATE_TITLE')" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0 pt-2">
                <v-btn dark color="secondary" @click="emits('request_clear_check_state')" hide-details>{{
                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-select class="select" v-model="check_state" :items="check_states" :label="i18n.global.t('CHECK_STATE_TITLE')"
            item-title="name" item-value="value" />
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { miExtructCheckStateQueryEmits } from './mi-extruct-check-state-query-emits'
import type { miExtructCheckStateQueryProps } from './mi-extruct-check-state-query-props'
import { useMiExtructCheckStateQuery } from '@/classes/use-mi-extruct-check-state-query'

const props = defineProps<miExtructCheckStateQueryProps>()
const emits = defineEmits<miExtructCheckStateQueryEmits>()

const {
    use_mi_check_state,
    check_state,
    check_states,
    get_update_extruct_check_state,
} = useMiExtructCheckStateQuery({ props, emits })

defineExpose({ get_update_extruct_check_state })
</script>
