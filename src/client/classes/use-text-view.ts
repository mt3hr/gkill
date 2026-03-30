import type { TextViewProps } from '@/pages/views/text-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useTextView(options: {
    props: TextViewProps,
    emits: KyouViewEmits,
}) {
    const { props: _props, emits: _emits } = options

    return {
    }
}
