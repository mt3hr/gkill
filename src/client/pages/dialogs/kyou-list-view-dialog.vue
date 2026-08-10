<template>
    <Teleport to="body" v-if="is_show_dialog" >
        <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

        <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog kyou-list-view-dialog"
            :class="ui.isTransparent.value ? 'is-transparent' : ''">
            <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
                @touchstart="ui.onHeaderPointerDown">
                <div class="gkill-floating-dialog__title"></div>
                <div class="gkill-floating-dialog__spacer"></div>
                <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
                    :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'" variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
            </div>

            <div class="gkill-floating-dialog__body">

                <!-- 件数表示以外の残り全部を占める。basis 0 + grow 1 で body の高さから素直に決まる -->
                <v-card v-if="is_show_dialog" class="kyou_list_view_dialog_view pa-2" style="flex: 1 1 0; min-height: 0; width: 100%;"
                    ref="list_card_ref">
                    <KyouListView :kyou_height="180" :width="view_width" :list_height="view_height"
                        :application_config="application_config" :gkill_api="gkill_api" :matched_kyous="model_value!"
                        :query="new FindKyouQuery()" :is_focused_list="true"
                        :closable="false" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                        :is_readonly_mi_check="true" :show_checkbox="true" :show_footer="false"
                        :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true" :show_content_only="false"
                        :show_timeis_plaing_end_button="false" :show_rep_name="show_rep_name"
                        :force_show_latest_kyou_info="force_show_latest_kyou_info"
                        ref="kyou_list_views" v-on="crudRelayHandlers" />
                </v-card>
                <!-- App.vue の .gkill-floating-dialog__body .v-card が flex: 1 1 auto を与えるので、
                     flex-shrink だけ潰すと grow が残り、件数表示がKyouListViewと領域を半分ずつ分け合ってしまう。
                     件数表示は必要な高さだけ取り、残りは全部KyouListViewに渡す -->
                <v-card variant="text" :ripple="false" :link="false" class="px-2" style="flex: 0 0 auto;">
                    <v-row no-gutters>
                        <v-col v-if="model_value && model_value.length" cols="auto" class="py-3">
                            {{ model_value.length }}{{ i18n.global.t("N_COUNT_ITEMS_TITLE") }}
                        </v-col>
                        <v-spacer />
                    </v-row>
                </v-card>
            </div>
        </div>
    </Teleport>

    <!-- このダイアログの中から開くrykvダイアログは、ページ最上位ではなくここでホストする。
         上へ持ち上げるとタグ追加等の requested_reload_kyou がページにしか届かず、
         このリストに戻ってこない（Vueのイベントは上方向にしか流れない）。
         Teleportの外に置くのは、中に入れるとリストダイアログを閉じた瞬間に
         子ダイアログが強制unmountされ、ダイアログ履歴の巻き戻しと競合するため。
         dialogs が空なら v-for は0回なのでDOMは一切生まない -->
    <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialogHostHandlers" />
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, type ComponentPublicInstance, ref, watch } from 'vue'
import KyouListView from '../views/kyou-list-view.vue';
import RykvDialogHost from '../views/rykv-dialog-host.vue';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import type { Kyou } from '@/classes/datas/kyou';
import type { KyouListViewDialogProps } from './kyou-list-view-dialog-props';
import type { KyouListViewEmits } from '../views/kyou-list-view-emits';
import { useKyouListViewDialog } from '@/classes/use-kyou-list-view-dialog';
import { i18n } from '@/i18n'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

// ルートが Teleport と RykvDialogHost の2つになるので、props に無い属性
// (kyou_height / width 等。中の KyouListView 用に呼び出し元が付けている) の
// 自動継承ができず警告になる。元々どこにも当てていない属性なので継承を切る
defineOptions({ inheritAttrs: false })

const props = defineProps<KyouListViewDialogProps>()
const model_value = defineModel<Array<Kyou>>()
const emits = defineEmits<KyouListViewEmits>()

const {
    // State
    is_show_dialog,
    opened_dialogs,

    // Business logic
    show,
    hide,

    // Event relay objects
    crudRelayHandlers,
    dialogHostHandlers,
} = useKyouListViewDialog({ props, emits, model_value })

defineExpose({ show, hide })

const ui = useFloatingDialog("kyou-list-view-dialog", {
  centerMode: "always",
  onEscape: () => hide(),
})

// ダイアログはユーザ操作でリサイズされる。useFloatingDialog は外側コンテナに
// inline width/height を書くだけで子には通知しないので、リストを載せている
// v-card の実寸を ResizeObserver で測って KyouListView に px で渡す。
// (KyouListView は v-virtual-scroll(renderless) の表示行数計算に数値の高さが要るため、
//  CSS の flex 追従だけでは埋まらない)
const list_card_ref = ref<ComponentPublicInstance | HTMLElement | null>(null)
const observed_width = ref(0)
const observed_height = ref(0)

// kyou-list-view.vue はスクロールコンテナを width + 8 で描くので、その分を差し引く
const view_width = computed(() => observed_width.value > 0 ? Math.max(200, observed_width.value - 8) : 400)
const view_height = computed(() => observed_height.value > 0 ? observed_height.value : props.list_height.valueOf())

function resolve_element(target: ComponentPublicInstance | HTMLElement | null): HTMLElement | null {
    if (!target) return null
    return target instanceof HTMLElement ? target : (target.$el as HTMLElement | null)
}

let card_ro: ResizeObserver | null = null
watch(list_card_ref, (el, old_el) => {
    const old_element = resolve_element(old_el ?? null)
    if (card_ro && old_element) { try { card_ro.unobserve(old_element) } catch { /* noop */ } }
    const element = resolve_element(el)
    if (element) {
        if (!card_ro) {
            card_ro = new ResizeObserver((entries) => {
                for (const entry of entries) {
                    observed_width.value = entry.contentRect.width
                    observed_height.value = entry.contentRect.height
                }
            })
        }
        card_ro.observe(element)
    }
}, { flush: 'post' })
onBeforeUnmount(() => { card_ro?.disconnect(); card_ro = null })
</script>

<style scoped lang="css">
.kyou_list_view_dialog_view,
.kyou_list_view_dialog {
    overflow-y: hidden !important;
}

.kyou_detail_view,
.kyou_list_view,
.v-dialog .v-card {
    overflow-y: hidden !important;
}

.kyou_list_view_dialog_view :deep(.kyou_list_view) {
    width: 100% !important;
}

/* App.vue の .gkill-floating-dialog__body .v-card { overflow: auto } が
   リスト内の各Kyouのカードにも当たり、1件ずつが縦スクロールしてしまうのを打ち消す。
   display:flex/column はそのまま残す (mi-re-kyou-view の .mirekyou_card が前提にしている) */
.kyou_list_view_dialog_view :deep(.kyou_in_list .v-card) {
    overflow: hidden !important;
    flex: 0 0 auto;
}

/* 各Kyouは v-card の width 属性で px 指定されるが、スクロールバーの有無で
   スクロールコンテナの内寸は変わる。CSSで100%に倒して常に幅いっぱいにする */
.kyou_list_view_dialog_view :deep(.kyou_in_list),
.kyou_list_view_dialog_view :deep(.kyou_in_list > *),
.kyou_list_view_dialog_view :deep(.kyou_in_list .v-card) {
    width: 100% !important;
    max-width: 100% !important;
}
</style>

<style lang="css">
/* Teleport 先では data-v-xxx が付かないため非スコープで定義 */
/* 未リサイズ時も高さを確定させる。高さがコンテンツ依存だとリストの実寸が測れない */
.kyou-list-view-dialog:not(.is-user-resized) {
    width: min(900px, 92vw);
    max-width: 92vw;
    height: 85vh;
}

.kyou-list-view-dialog:not(.is-user-resized) .gkill-floating-dialog__body {
    width: 100%;
    max-width: none;
    max-height: none;
    flex: 1 1 auto;
    min-height: 0;
}

/* App.vue の is-user-resized 用ルールには min-height: 0 が無く、body が縮まない */
.kyou-list-view-dialog.is-user-resized .gkill-floating-dialog__body {
    max-height: none;
    min-height: 0;
}
</style>
