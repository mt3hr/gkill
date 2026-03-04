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
      <v-btn dark color="primary" class="pa-3 ma-3" v-for="template, index in template.children" :key="template.id!"
        @click="clicked_template_button(template, index)">
        {{ template.title }}

        <KFTLTemplateDialog :application_config="application_config" :gkill_api="gkill_api" :template="template"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @clicked_template_element_leaf="(...t: any[]) => emits('clicked_template_element_leaf', t[0] as KFTLTemplateElementData)"
          @requested_close_dialog="emits('requested_close_dialog')" ref="child_template_dialogs" />
      </v-btn>
    </div>
  </v-card>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import KFTLTemplateDialog from '../dialogs/kftl-template-dialog.vue'
import type { KFTLTemplateViewProps } from './kftl-template-view-props'
import type { KFTLTemplateViewEmits } from './kftl-template-view-emits'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const child_template_dialogs: Ref<Array<any>> = ref(new Array<any>())

defineProps<KFTLTemplateViewProps>()
const emits = defineEmits<KFTLTemplateViewEmits>()

function clicked_template_button(template: KFTLTemplateElementData, index: number): void {
  if (!template.children) {
    emits('clicked_template_element_leaf', template)
    emits('requested_close_dialog')
    return
  }
  child_template_dialogs.value[index].show()
}
</script>
