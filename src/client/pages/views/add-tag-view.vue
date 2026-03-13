<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ADD_TAG_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field v-model="tag_name" :label="i18n.global.t('TAG_TITLE')" autofocus
            :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{
                    i18n.global.t('SAVE_TITLE')
                }}</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :is_image_request_to_thumb_size="false" :highlight_targets="highlight_targets" :is_image_view="false"
                :kyou="kyou" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :is_readonly_mi_check="false" :show_attached_timeis="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_update_time="false"
                :show_related_time="true" :show_attached_tags="true" :show_attached_texts="true"
                :show_attached_notifications="true"
                v-on="crudRelayHandlers" />
        </v-card>
    </v-card>
    <Teleport to="body" v-if="show_confirm_unknown_tag_dialog">
        <div class="gkill-float-scrim" :class="confirm_dialog_ui.isTransparent.value ? 'is-transparent' : ''" />
        <div :ref="confirm_dialog_ui.containerRef" :style="confirm_dialog_ui.fixedStyle.value"
            class="gkill-floating-dialog"
            :class="confirm_dialog_ui.isTransparent.value ? 'is-transparent' : ''">
            <div class="gkill-floating-dialog__header pa-0 ma-0"
                @mousedown="confirm_dialog_ui.onHeaderPointerDown"
                @touchstart="confirm_dialog_ui.onHeaderPointerDown">
                <div class="gkill-floating-dialog__title"></div>
                <div class="gkill-floating-dialog__spacer"></div>
                <v-checkbox v-model="confirm_dialog_ui.isTransparent.value" color="white" size="small" variant="flat"
                    :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="cancel_save()" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-close</v-icon>
                </v-btn>
            </div>
            <div class="gkill-floating-dialog__body">
                <v-card class="pa-2">
                    <v-card-title>
                        <v-row class="pa-0 ma-0">
                            <v-col cols="auto" class="pa-0 ma-0">
                                <span>{{ i18n.global.t("CONFIRM_UNKNOWN_TAG_TITLE") }}</span>
                            </v-col>
                        </v-row>
                    </v-card-title>
                    <v-card-text>
                        {{ i18n.global.t("CONFIRM_UNKNOWN_TAG_MESSAGE") }}
                        <v-list density="compact">
                            <v-list-item v-for="tag in unknown_tags" :key="tag">
                                <v-list-item-title>{{ tag }}</v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-card-text>
                    <v-row class="pa-0 ma-0">
                        <v-spacer />
                        <v-col cols="auto" class="pa-0 ma-0">
                            <v-btn @click="cancel_save()">{{ i18n.global.t("CANCEL_TITLE") }}</v-btn>
                        </v-col>
                        <v-col cols="auto" class="pa-0 ma-0">
                            <v-btn dark color="primary" @click="confirm_save()">{{ i18n.global.t("SAVE_TITLE")
                                }}</v-btn>
                        </v-col>
                    </v-row>
                </v-card>
            </div>
        </div>
    </Teleport>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AddTagViewProps } from './add-tag-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { useAddTagView } from '@/classes/use-add-tag-view'

const props = defineProps<AddTagViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // State
    is_requested_submit,
    show_kyou,
    tag_name,
    show_confirm_unknown_tag_dialog,
    unknown_tags,

    // Dialog UI
    confirm_dialog_ui,

    // Business logic / template handlers
    save,
    cancel_save,
    confirm_save,

    // Event relay objects
    crudRelayHandlers,
} = useAddTagView({ props, emits })
</script>
