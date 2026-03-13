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
import type { LantanaFlowersViewEmits } from './lantana-flowers-view-emits'
import type { LantanaFlowersViewProps } from './lantana-flowers-view-props'
import LantanaFlower from './lantana-flower.vue'
import { useLantanaFlowersView } from '@/classes/use-lantana-flowers-view'

const props = defineProps<LantanaFlowersViewProps>()
const emits = defineEmits<LantanaFlowersViewEmits>()

const {
    flower_state_1,
    flower_state_2,
    flower_state_3,
    flower_state_4,
    flower_state_5,
    set_mood,
    get_mood,
} = useLantanaFlowersView({ props, emits })

defineExpose({ set_mood, get_mood })
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