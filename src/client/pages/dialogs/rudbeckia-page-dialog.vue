<template>
    <Teleport to="body" v-if="is_show_dialog">
        <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

        <div :ref="ui.containerRef" :style="ui.fixedStyle.value"
            class="gkill-floating-dialog rudbeckia-page-dialog"
            :class="ui.isTransparent.value ? 'is-transparent' : ''"
            data-gkill-non-blocking="true">
            <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
                @touchstart="ui.onHeaderPointerDown">
                <div class="gkill-floating-dialog__title"></div>
                <div class="gkill-floating-dialog__spacer"></div>
                <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
                    :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="help_dialog?.show()" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-help-circle-outline</v-icon>
                </v-btn>
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-close</v-icon>
                </v-btn>
            </div>

            <div class="gkill-floating-dialog__body"
                :ref="(el: Element | ComponentPublicInstance | null) => { dialog_body_ref = el as HTMLElement | null }">
                <!--
                  ホストしたビューは自前の v-app-bar / v-navigation-drawer / v-main を持っている。
                  <v-layout> で包むと Vuetify がそれらを「入れ子レイアウト」として扱い、
                  position: fixed ではなく position: absolute でこの箱の中へ配置する
                  (vuetify/lib/composables/layout.js:94,211,262)。包み忘れると画面最上部へ飛ぶ。
                  実行中(plaing)はレイアウト部品を持たないので包まない。
                -->
                <v-layout v-if="kind !== 'plaing'" class="rudbeckia-hosted-layout"
                    :height="layout_height" :width="layout_width">
                    <RykvView v-if="kind === 'rykv'"
                        :app_content_height="view_height" :app_content_width="view_width"
                        :app_title_bar_height="HOSTED_APP_BAR_HEIGHT"
                        :application_config="application_config" :gkill_api="gkill_api"
                        :is_shared_rykv_view="false" :share_title="''"
                        :application_config_load_failed="application_config_load_failed"
                        :column_state_instance_key="column_state_instance_key"
                        :is_hosted_in_dialog="true" :kyou_change_channel="kyou_change_channel"
                        v-on="hostedViewHandlers" />
                    <MiView v-else-if="kind === 'mi'"
                        :app_content_height="view_height" :app_content_width="view_width"
                        :app_title_bar_height="HOSTED_APP_BAR_HEIGHT"
                        :application_config="application_config" :gkill_api="gkill_api"
                        :application_config_load_failed="application_config_load_failed"
                        :column_state_instance_key="column_state_instance_key"
                        :is_hosted_in_dialog="true" :kyou_change_channel="kyou_change_channel"
                        v-on="hostedViewHandlers" />
                    <DashboardView v-else-if="kind === 'dashboard'"
                        :app_content_height="view_height" :app_content_width="view_width"
                        :app_title_bar_height="HOSTED_APP_BAR_HEIGHT"
                        :application_config="application_config" :gkill_api="gkill_api"
                        :application_config_load_failed="application_config_load_failed"
                        :is_hosted_in_dialog="true" :kyou_change_channel="kyou_change_channel"
                        v-on="hostedViewHandlers" />
                </v-layout>
                <PlaingTimeIsView v-else
                    :app_content_height="layout_height" :app_content_width="layout_width"
                    :application_config="application_config" :gkill_api="gkill_api"
                    :is_hosted_in_dialog="true" :kyou_change_channel="kyou_change_channel"
                    v-on="hostedViewHandlers" />
                <HelpDialog :screen_name="help_screen_name" ref="help_dialog" />
            </div>
        </div>
    </Teleport>
</template>

<script lang="ts" setup>
import { ref, type ComponentPublicInstance } from 'vue'
import { i18n } from '@/i18n'
import RykvView from '../views/rykv-view.vue'
import MiView from '../views/mi-view.vue'
import DashboardView from '../views/dashboard-view.vue'
import PlaingTimeIsView from '../views/plaing-time-is-view.vue'
import HelpDialog from './help-dialog.vue'
import type { RudbeckiaPageDialogProps } from './rudbeckia-page-dialog-props'
import type { RudbeckiaPageDialogEmits } from './rudbeckia-page-dialog-emits'
import { useRudbeckiaPageDialog, HOSTED_APP_BAR_HEIGHT } from '@/classes/use-rudbeckia-page-dialog'
import { build_rudbeckia_hosted_view_relay } from '@/classes/rudbeckia-hosted-view-relay'

const props = defineProps<RudbeckiaPageDialogProps>()
const emits = defineEmits<RudbeckiaPageDialogEmits>()

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)

const {
    is_show_dialog,
    ui,
    dialog_body_ref,
    column_state_instance_key,
    kyou_change_channel,
    layout_width,
    layout_height,
    view_width,
    view_height,
    help_screen_name,
    show,
    hide,
} = useRudbeckiaPageDialog({ props, emits })

// 中のビューが出す17件をそのまま上げ、画面切替だけポート向けに読み替える
const hostedViewHandlers = build_rudbeckia_hosted_view_relay(emits)

defineExpose({ show, hide })
</script>

<style lang="css">
/* Teleport 先では data-v-xxx が付かないため非スコープで定義 */
/* 未リサイズ時も幅と高さを確定させる。確定していないと中のビューへ渡す実寸が測れないし、
   既定の .gkill-floating-dialog__body { width: min(500px, 85vw) } のままだと
   318px のサイドバーの隣に182pxしか残らない */
.rudbeckia-page-dialog:not(.is-user-resized) {
    width: min(1200px, 92vw);
    max-width: 92vw;
    height: 88vh;
}

.rudbeckia-page-dialog:not(.is-user-resized) .gkill-floating-dialog__body {
    width: 100%;
    max-width: none;
    max-height: none;
    flex: 1 1 auto;
    min-height: 0;
}

/* App.vue の is-user-resized 用ルールには min-height: 0 が無く、body が縮まない */
.rudbeckia-page-dialog.is-user-resized .gkill-floating-dialog__body {
    max-height: none;
    min-height: 0;
}

/* 列は素の table なので、単独ページでは body の overflow-x が横あふれを受けている。
   ダイアログの body は overflow-x: hidden なので、ここで受け直さないと列が切れて出せない */
.rudbeckia-page-dialog .gkill-floating-dialog__body {
    overflow-x: hidden;
}

/* ここに載っているのは「画面まるごと」なので、App.vue の
   `.gkill-floating-dialog__body .v-card`（＝中身はカード1枚、という前提の規則）は
   一枚残らず邪魔になる。中の v-card は全部、単独ページと同じ Vuetify の既定へ戻す。
   戻さないとサイドバーの節も一覧の行も個別のスクロール箱になる */
.rudbeckia-page-dialog .gkill-floating-dialog__body .v-card {
    display: block;
    flex-direction: unset;
    overflow: hidden;
    flex: 0 1 auto;
}

.rudbeckia-page-dialog .rudbeckia-hosted-layout {
    overflow: hidden;
}

/* 入れ子レイアウトで position: absolute になったアプリバー/サイドバーの基準は
   <v-layout> ではなく「最も近い位置指定済み祖先」＝ビューのルート。
   箱いっぱいに重ねておかないとバーがずれる。
   .dashboard_view_wrap / .plaing_timeis_view_wrap は position: relative を持たないので
   ここで付ける（App.vue:96-99 が持っているのは rykv / mi / saihate だけ） */
.rudbeckia-page-dialog .rudbeckia-hosted-layout > * {
    position: relative;
    width: 100%;
    height: 100%;
}

/* v-main はアプリバーとサイドバーの外側。ここで横スクロールを受ける。
   FAB は position: absolute でビューのルートが基準なので、一緒にはスクロールしない */
.rudbeckia-page-dialog .rudbeckia-hosted-layout .v-main {
    overflow-x: auto;
    min-width: 0;
}
</style>
