<!-- テキスト中の URL をリンク化して表示する。
     親要素の white-space: pre-line / pre-wrap をそのまま活かすため、
     テンプレートはセグメント間に空白・改行を入れない1行書きにすること（整形すると表示に空白が混入する）。 -->
<template><template v-for="(segment, index) in segments" :key="index"><a v-if="segment.is_url" :href="segment.text" target="_blank" rel="noopener noreferrer" @click.stop>{{ segment.text }}</a><template v-else>{{ segment.text }}</template></template></template>
<script setup lang="ts">
// @click.stop は必須。親の kyou-view.vue が @click.prevent を持つため、
// stop しないとバブルした click に preventDefault が掛かりリンクが開かない。
import type { LinkifiedTextProps } from './linkified-text-props'
import { useLinkifiedText } from '@/classes/use-linkified-text'

const props = defineProps<LinkifiedTextProps>()

const {
    segments,
} = useLinkifiedText({ props })
</script>
