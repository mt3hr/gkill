import { type Ref, nextTick, ref, watch } from 'vue'
import { Text } from '@/classes/datas/text'
import type { TextHistoriesViewProps } from '@/pages/views/text-histories-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useTextHistoriesView(options: {
    props: TextHistoriesViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    const cloned_text: Ref<Text> = ref(props.text.clone())
    watch(() => props.text, () => {
        cloned_text.value = props.text.clone()
        nextTick(() => cloned_text.value.load_attached_histories())
    })
    nextTick(() => cloned_text.value.load_attached_histories())

    return {
        cloned_text,
    }
}
