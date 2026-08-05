<template>
    <span v-if="timeis_kyou.typed_timeis" :class="plaing_class"
        @contextmenu.prevent="async (e: PointerEvent) => show_context_menu(e)"
        @dblclick.stop.prevent="show_kyou_dialog">
        <span class="plaing_label">{{ timeis_kyou.typed_timeis.title }}</span>
    </span>
    <AttachedTimeIsPlaingContextMenu :application_config="application_config" :gkill_api="gkill_api" :target_kyou="kyou"
        v-if="timeis_kyou.typed_timeis" :timeis_kyou="timeis_kyou"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :highlight_targets="highlight_targets"
        @deleted_kyou="(deleted_kyou: Kyou) => emits('deleted_kyou', deleted_kyou)"
        @deleted_tag="(deleted_tag: Tag) => emits('deleted_tag', deleted_tag)"
        @deleted_text="(deleted_text: Text) => emits('deleted_text', deleted_text)"
        @deleted_notification="(deleted_notification: Notification) => emits('deleted_notification', deleted_notification)"
        @registered_kyou="(registered_kyou: Kyou) => emits('registered_kyou', registered_kyou)"
        @registered_tag="(registered_tag: Tag) => emits('registered_tag', registered_tag)"
        @registered_text="(registered_text: Text) => emits('registered_text', registered_text)"
        @registered_notification="(registered_notification: Notification) => emits('registered_notification', registered_notification)"
        @updated_kyou="(updated_kyou: Kyou) => emits('updated_kyou', updated_kyou)"
        @updated_tag="(updated_tag: Tag) => emits('updated_tag', updated_tag)"
        @updated_text="(updated_text: Text) => emits('updated_text', updated_text)"
        @updated_notification="(updated_notification: Notification) => emits('updated_notification', updated_notification)"
        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
        @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous: Kyou[], checked: boolean) => emits('requested_update_check_kyous', kyous, checked)"
            @requested_open_rykv_dialog="(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload)"
        ref="context_menu" />
</template>
<script setup lang="ts">
import type { AttachedTimeIsPlaingProps } from './attached-time-is-plaing-props'
import type { RykvDialogKind, RykvDialogPayload } from "./rykv-dialog-kind"
import type { KyouViewEmits } from './kyou-view-emits'
import AttachedTimeIsPlaingContextMenu from './attached-time-is-plaing-context-menu.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
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
