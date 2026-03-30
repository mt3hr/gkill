<template>
  <!-- NOTE:
    このコンポーネントは「View」なので、Teleport / is_show_dialog で包まない。
    表示/非表示は親（kftl-template-dialog.vue）側が制御する。
  -->
  <v-card :max-width="500" class="help_card pa-3">
    <v-card-title>
      <v-row>
        <v-col cols="auto" class="pa-0 ma-0">
          {{ i18n.global.t("KFTL_TEMPLATE_TITLE") }}
        </v-col>
        <v-spacer />
      </v-row>
    </v-card-title>

    <div class="button_list">
      <v-btn dark color="primary" class="pa-3 ma-3" v-for="(child, index) in template.children" :key="child.id!"
        @click="clicked_template_button(child, index)">
        {{ child.title }}

        <KFTLTemplateDialog :application_config="application_config" :gkill_api="gkill_api" :template="child"
          @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
          @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
          @clicked_template_element_leaf="(t: KFTLTemplateElementData) => emits('clicked_template_element_leaf', t)"
          @requested_close_dialog="emits('requested_close_dialog')" ref="child_template_dialogs" />
      </v-btn>
    </div>
  </v-card>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import KFTLTemplateDialog from '../dialogs/kftl-template-dialog.vue'
import type { KFTLTemplateViewProps } from './kftl-template-view-props'
import type { KFTLTemplateViewEmits } from './kftl-template-view-emits'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useKFTLTemplateView } from '@/classes/use-kftl-template-view'

const props = defineProps<KFTLTemplateViewProps>()
const emits = defineEmits<KFTLTemplateViewEmits>()

const {
    child_template_dialogs,
    clicked_template_button,
} = useKFTLTemplateView({ props, emits })
</script>
