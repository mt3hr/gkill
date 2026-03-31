<template>
    <v-card class="kftl_view">
        <v-card-title :height="title_height">
            <v-row>
                <v-col cols="auto">
                    {{ i18n.global.t("KFTL_ADD_KYOU_TITLE") }}
                </v-col>
                <v-spacer />
                <v-col cols="auto">
                    <v-btn dark color="primary" @click="show_kftl_template_dialog" :disabled="is_requested_submit">{{
                        i18n.global.t("KFTL_TEMPLATE_TITLE") }}</v-btn>
                </v-col>
                <v-col cols="auto">
                    <v-btn dark color="primary" @click="submit" :disabled="is_requested_submit">{{
                        i18n.global.t("SAVE_TITLE")
                    }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <table class="kftl_input">
            <tr>
                <td>
                    <div class="kftl_line_label line_label_wrap">
                        <KFTLLineLabel v-for="(line_label_data, index) in line_label_datas"
                            :key="index" :application_config="application_config"
                            :gkill_api="gkill_api" :line_label_data="line_label_data"
                            :style="line_label_styles[index]" />
                    </div>
                </td>
                <td>
                    <div class="kftl_text_area_wrap">
                        <textarea id="kftl_text_area" class="kftl_text_area" v-model="text_area_content"
                            :readonly="is_requested_submit" autofocus></textarea>
                    </div>
                </td>
            </tr>
        </table>
        <KFTLTemplateDialog :application_config="application_config" :gkill_api="gkill_api"
            :template="application_config.kftl_template_struct"
            v-on="errorMessageRelayHandlers"
            @clicked_template_element_leaf="paste_template" ref="kftl_template_dialog" />
    </v-card>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'

import type { KFTLProps } from './kftl-props'
import type { KFTLViewEmits } from './kftl-view-emits'

import KFTLLineLabel from './kftl-line-label.vue'
import KFTLTemplateDialog from '../dialogs/kftl-template-dialog.vue'
import { useKftlView } from '@/classes/use-kftl-view'

const props = defineProps<KFTLProps>()
const emits = defineEmits<KFTLViewEmits>()

const {
    // Template refs
    kftl_template_dialog,

    // State
    text_area_content,
    line_label_datas,
    line_label_styles,
    is_requested_submit,
    title_height,

    // Computed
    text_area_width_px,
    text_area_height_px,
    line_label_width_px,
    line_label_height_px,
    kftl_input_height_px,
    kftl_input_width_px,

    // Business logic
    submit,
    show_kftl_template_dialog,
    paste_template,
    focus_kftl_text_area,

    // Event relay objects
    errorMessageRelayHandlers,
} = useKftlView({ props, emits })

defineExpose({ focus_kftl_text_area })
</script>

<style lang="css" scoped>
.kftl_text_area_wrap {
    height: 100%;
    width: calc(v-bind(text_area_width_px));
}

.kftl_text_area {
    height: calc(v-bind(text_area_height_px));
    width: calc(v-bind(text_area_width_px));
    resize: none;
    font-size: 1em;
    line-height: 24px;
}

.line_label_wrap {
    color: rgb(var(--v-theme-background-focused));
    padding-right: 16px;
    height: calc(v-bind(line_label_height_px));
    width: calc(v-bind(line_label_width_px));
    text-align: right;
}

textarea {
    border: solid 1px silver;
}

.kftl_input {
    height: calc(v-bind(kftl_input_height_px));
    width: calc(v-bind(kftl_input_width_px));
    overflow-y: scroll;
}

.kftl_line_label {
    overflow-y: hidden;
}

.kftl_line_label::-webkit-scrollbar {
    display: none;
}
</style>
