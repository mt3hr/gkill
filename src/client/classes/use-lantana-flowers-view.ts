'use strict'

import { type Ref, ref, watch } from 'vue'
import type { LantanaFlowersViewProps } from '@/pages/views/lantana-flowers-view-props'
import type { LantanaFlowersViewEmits } from '@/pages/views/lantana-flowers-view-emits'
import { LantanaFlowerState } from '@/classes/lantana/lantana-flower-state'

export function useLantanaFlowersView(options: { props: LantanaFlowersViewProps, emits: LantanaFlowersViewEmits }) {
    const { props, emits } = options

    const mood: Ref<number> = ref(props.mood)

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

    async function get_mood(): Promise<number> {
        return mood.value
    }

    async function set_mood(mood_value: number): Promise<void> {
        if (!props.editable) {
            return
        }
        mood.value = mood_value
    }

    async function emit_updated_mood(): Promise<void> {
        emits('updated_mood', mood.value)
    }

    return {
        mood,
        flower_state_1,
        flower_state_2,
        flower_state_3,
        flower_state_4,
        flower_state_5,
        get_mood,
        set_mood,
    }
}
