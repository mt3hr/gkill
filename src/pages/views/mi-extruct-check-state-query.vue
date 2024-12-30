<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="use_mi_check_state" readonly hide-details class="pa-0 ma-0" label="チェック状態" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0">
                <v-btn @click="emits('request_clear_check_state')" hide-details>クリア</v-btn>
            </v-col>
        </v-row>
        <v-select class="select" v-model="check_state" :items="check_states" :label="'チェック状態'" item-title="name"
            item-value="value" />
    </div>
</template>
<script lang="ts" setup>
import { nextTick, type Ref, ref, watch } from 'vue'
import { MiCheckState } from '@/classes/api/find_query/mi-check-state'
import type { miExtructCheckStateQueryEmits } from './mi-extruct-check-state-query-emits'
import type { miExtructCheckStateQueryProps } from './mi-extruct-check-state-query-props'
import type { CheckState } from './check-state';
const props = defineProps<miExtructCheckStateQueryProps>()
const emits = defineEmits<miExtructCheckStateQueryEmits>()
const query = ref(props.find_kyou_query.clone())
defineExpose({ get_update_extruct_check_state })

watch(() => props.find_kyou_query, () => {
    query.value = props.find_kyou_query.clone()
    load_check_state()
})

nextTick(() => {
    load_check_state()
    emits('inited')
})
const use_mi_check_state = ref(true)
const check_state: Ref<MiCheckState> = ref(MiCheckState.uncheck)

watch(() => check_state.value, () => {
    console.log(check_state)
    emits('request_update_extruct_check_state', check_state.value)
})

function load_check_state(): void {
    for (let i = 0; i < check_states.value.length; i++) {
        if (query.value.mi_check_state === check_states.value[i].value) {
            check_state.value = check_states.value[i].value
            break
        }
    }
}

const check_states: Ref<Array<{ name: string, value: MiCheckState }>> = ref([
    {
        name: "すべて",
        value: MiCheckState.all,
    },
    {
        name: "チェック済み",
        value: MiCheckState.checked,
    },
    {
        name: "未チェック",
        value: MiCheckState.uncheck,
    }
])

function get_update_extruct_check_state(): MiCheckState {
    return check_state.value
}
</script>
