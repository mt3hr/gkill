<template>
    <span v-if="timeis_kyou.typed_timeis" :class="plaing_class"
        @contextmenu.prevent="async (e: PointerEvent) => show_context_menu(e)"
        @dblclick.stop.prevent="show_kyou_dialog">
        <span class="plaing_label">{{ timeis_kyou.typed_timeis.title }}</span>
    </span>
    <AttachedTimeIsPlaingContextMenu :application_config="application_config" :gkill_api="gkill_api" :target_kyou="kyou"
        v-if="timeis_kyou.typed_timeis" :timeis_kyou="timeis_kyou"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :highlight_targets="highlight_targets"
        v-on="crudRelayHandlers"
        ref="context_menu" />
</template>
<script setup lang="ts">
import type { AttachedTimeIsPlaingProps } from './attached-time-is-plaing-props'
import type { KyouViewEmits } from './kyou-view-emits'
import AttachedTimeIsPlaingContextMenu from './attached-time-is-plaing-context-menu.vue'
import { useAttachedTimeIsPlaing } from '@/classes/use-attached-time-is-plaing'

const props = defineProps<AttachedTimeIsPlaingProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // Template refs
    context_menu,

    // State
    plaing_class,

    // Methods used in template
    show_context_menu,
    show_kyou_dialog,
    // Event relay objects
    crudRelayHandlers,
} = useAttachedTimeIsPlaing({ props, emits })
</script>
<style scoped>
/* 形は ◀[本文]▶ (本文が空ならひし形)。左右の三角は擬似要素のborderで作る。
   従来は「親全体をlightgrayで塗り、三角の切り欠きをテーマ背景色で上塗りする」
   作りだったため、切り欠きがダークテーマでは黒く、ハイライト(緑)の上でも
   黒く浮いていた。親は透過にして本文(.plaing_label)だけを塗ることで、
   切り欠きを本当に透明にする。三角の寸法は従来のまま。 */
.plaing,
.highlighted_plaing {
    /* タグとの合わせ */
    position: relative;
    display: inline-flex;
    border: solid transparent 2px;
    border-left: 0px;
    color: blue;
    cursor: pointer;
    font-size: small;
    background: transparent;
}

.plaing::after,
.highlighted_plaing::after,
.plaing::before,
.highlighted_plaing::before {
    content: "";
    background: transparent;
    border-top: 9.5px solid transparent;
    border-bottom: 9.5px solid transparent;
}

.plaing .plaing_label {
    background: lightgray;
}

.plaing::after {
    border-left: 10px solid lightgray;
}

.plaing::before {
    border-right: 10px solid lightgray;
}

/* 選択時は塗りがハイライト色になり親(緑)と同化して形が見えなくなるので、
   通常時の塗り色で輪郭を描く。ひし形は矩形のborderではなぞれないため、
   レイアウトに影響しないfilterでシルエットそのものを縁取る(サイズ不変) */
.highlighted_plaing {
    filter: drop-shadow(1px 0 0 lightgray) drop-shadow(-1px 0 0 lightgray) drop-shadow(0 1px 0 lightgray) drop-shadow(0 -1px 0 lightgray);
}

.highlighted_plaing .plaing_label {
    background: rgb(var(--v-theme-highlight));
}

.highlighted_plaing::after {
    border-left: 10px solid rgb(var(--v-theme-highlight));
}

.highlighted_plaing::before {
    border-right: 10px solid rgb(var(--v-theme-highlight));
}
</style>
