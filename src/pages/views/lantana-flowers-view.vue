<template>
    <table>
        <tr class="lantana_icon_tr">
            <td class="lantana_icon_td">
                <LantanaFlower :application_config="application_config" :editable="editable" :gkill_api="gkill_api"
                    :state="flower_state_1" @clicked_left="set_mood(1)" @clicked_right="set_mood(2)" />
            </td>
            <td class="lantana_icon_td">
                <LantanaFlower :application_config="application_config" :editable="editable" :gkill_api="gkill_api"
                    :state="flower_state_2" @clicked_left="set_mood(3)" @clicked_right="set_mood(4)" />
            </td>
            <td class="lantana_icon_td">
                <LantanaFlower :application_config="application_config" :editable="editable" :gkill_api="gkill_api"
                    :state="flower_state_3" @clicked_left="set_mood(5)" @clicked_right="set_mood(6)" />
            </td>
            <td class="lantana_icon_td">
                <LantanaFlower :application_config="application_config" :editable="editable" :gkill_api="gkill_api"
                    :state="flower_state_4" @clicked_left="set_mood(7)" @clicked_right="set_mood(8)" />
            </td>
            <td class="lantana_icon_td">
                <LantanaFlower :application_config="application_config" :editable="editable" :gkill_api="gkill_api"
                    :state="flower_state_5" @clicked_left="set_mood(9)" @clicked_right="set_mood(10)" />
            </td>
        </tr>
    </table>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { type Ref, ref, watch } from 'vue'
import type { LantanaFlowersViewEmits } from './lantana-flowers-view-emits'
import type { LantanaFlowersViewProps } from './lantana-flowers-view-props'
import LantanaFlower from './lantana-flower.vue'
import { LantanaFlowerState } from '@/classes/lantana/lantana-flower-state'

const props = defineProps<LantanaFlowersViewProps>()
const emits = defineEmits<LantanaFlowersViewEmits>()
defineExpose({ set_mood, get_mood })

const mood: Ref<Number> = ref(props.mood)

const flower_state_1: Ref<LantanaFlowerState> = ref(mood.value.valueOf() >= 2 ? LantanaFlowerState.full : (mood.value.valueOf() >= 1 ? LantanaFlowerState.half : LantanaFlowerState.none))
const flower_state_2: Ref<LantanaFlowerState> = ref(mood.value.valueOf() >= 4 ? LantanaFlowerState.full : (mood.value.valueOf() >= 3 ? LantanaFlowerState.half : LantanaFlowerState.none))
const flower_state_3: Ref<LantanaFlowerState> = ref(mood.value.valueOf() >= 6 ? LantanaFlowerState.full : (mood.value.valueOf() >= 5 ? LantanaFlowerState.half : LantanaFlowerState.none))
const flower_state_4: Ref<LantanaFlowerState> = ref(mood.value.valueOf() >= 8 ? LantanaFlowerState.full : (mood.value.valueOf() >= 7 ? LantanaFlowerState.half : LantanaFlowerState.none))
const flower_state_5: Ref<LantanaFlowerState> = ref(mood.value.valueOf() >= 10 ? LantanaFlowerState.full : (mood.value.valueOf() >= 9 ? LantanaFlowerState.half : LantanaFlowerState.none))

watch(() => props.mood, () => {
    mood.value = props.mood
})

watch(() => mood.value, () => {
    flower_state_1.value = (mood.value.valueOf() >= 2 ? LantanaFlowerState.full : (mood.value.valueOf() >= 1 ? LantanaFlowerState.half : LantanaFlowerState.none))
    flower_state_2.value = (mood.value.valueOf() >= 4 ? LantanaFlowerState.full : (mood.value.valueOf() >= 3 ? LantanaFlowerState.half : LantanaFlowerState.none))
    flower_state_3.value = (mood.value.valueOf() >= 6 ? LantanaFlowerState.full : (mood.value.valueOf() >= 5 ? LantanaFlowerState.half : LantanaFlowerState.none))
    flower_state_4.value = (mood.value.valueOf() >= 8 ? LantanaFlowerState.full : (mood.value.valueOf() >= 7 ? LantanaFlowerState.half : LantanaFlowerState.none))
    flower_state_5.value = (mood.value.valueOf() >= 10 ? LantanaFlowerState.full : (mood.value.valueOf() >= 9 ? LantanaFlowerState.half : LantanaFlowerState.none))
    emit_updated_mood()
})

async function get_mood(): Promise<Number> {
    return mood.value
}
async function set_mood(mood_value: Number): Promise<void> {
    if (!props.editable) {
        return
    }
    mood.value = mood_value
}
async function emit_updated_mood(): Promise<void> {
    emits('updated_mood', mood.value)
}
</script>

<style scoped>
.lantana_icon_tr {
    width: calc(50px * 5);
    max-width: calc(50px * 5);
    min-width: calc(50px * 5);
}

.lantana_icon_td {
    width: 50px;
    height: 50px;
    max-width: 50px;
    min-width: 50px;
    max-height: 50px;
    min-height: 50px;
    display: inline-block;
}
</style>