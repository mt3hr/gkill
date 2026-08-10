import { computed } from 'vue'
import { split_text_by_urls } from '@/classes/linkify-text'
import type { LinkifiedTextProps } from '@/pages/views/linkified-text-props'

export function useLinkifiedText(options: {
    props: LinkifiedTextProps,
}) {
    const { props } = options

    // ── Computed ──
    const segments = computed(() => split_text_by_urls(props.text))

    // ── Return ──
    return {
        segments,
    }
}
