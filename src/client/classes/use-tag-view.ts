import type { TagViewProps } from '@/pages/views/tag-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useTagView(options: {
    props: TagViewProps,
    emits: KyouViewEmits,
}) {
    const { props: _props, emits: _emits } = options

    return {
    }
}
