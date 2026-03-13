import { type Ref, ref, watch } from 'vue'
import { Tag } from '@/classes/datas/tag'
import type { TagHistoriesViewProps } from '@/pages/views/tag-histories-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useTagHistoriesView(options: {
    props: TagHistoriesViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    const cloned_tag: Ref<Tag> = ref(props.tag.clone())
    watch(() => props.tag, () => {
        cloned_tag.value = props.tag.clone()
        cloned_tag.value.load_attached_histories()
    })
    cloned_tag.value.load_attached_histories()

    return {
        cloned_tag,
    }
}
