<template>
    <v-card elevation="0" class="kyou_list_view_card_wrap" :ripple="false" :link="false" @click.prevent="onClickedListView">
        <v-card elevation="0" class="kyou_list_view_card" :ripple="false" :link="false">
            <v-overlay v-model="is_loading" class="align-center justify-center" contained persistent>
                <v-progress-circular indeterminate color="primary" />
            </v-overlay>
            <v-virtual-scroll v-if="!query.is_image_only" :id="query.query_id.concat('_kyou_list_view')"
                class="kyou_list_view" :items="matched_kyous" :item-height="kyou_height_px"
                :height="list_height.valueOf() - footer_height.valueOf()" :width="width.valueOf() + 8"
                ref="kyou_list_view"
                @scrollend.prevent="onScrollEnd">
                <template v-slot:default="{ item }">
                    <KyouView class="kyou_in_list" :application_config="application_config" :gkill_api="gkill_api"
                        :draggable="draggable" :key="item.id" :highlight_targets="[]" :is_image_view="false"
                        :kyou="item" :show_checkbox="show_checkbox"
                        :show_content_only="show_content_only" :show_mi_create_time="true"
                        :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                        :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="show_timeis_plaing_end_button"
                        :width="width.valueOf()" :show_attached_timeis="false"
                        :is_readonly_mi_check="is_readonly_mi_check" :enable_context_menu="enable_context_menu"
                        :show_rep_name="show_rep_name" :force_show_latest_kyou_info="force_show_latest_kyou_info"
                        :show_update_time="false" :is_image_request_to_thumb_size="true"
                        :show_related_time="!(query.for_mi && item.data_type === 'mi_create' && (query.include_start_mi || query.include_end_mi || query.include_limit_mi))"
                        :enable_dialog="enable_dialog"
                        :height="kyou_height.valueOf()" :show_attached_tags="application_config.show_tags_in_list"
                        :show_attached_texts="false" :show_attached_notifications="false"
                        @focused_kyou="onFocusedKyou"
                        @clicked_kyou="onClickedKyou"
                        v-on="crudRelayHandlers" />
                </template>
            </v-virtual-scroll>
            <v-virtual-scroll v-if="query.is_image_only" :id="query.query_id.concat('_kyou_image_list_view')"
                class="kyou_list_view_image" :items="match_kyous_for_image" :item-height="kyou_height_px"
                :height="list_height.valueOf() - footer_height.valueOf()"
                :width="(200 * application_config.rykv_image_list_column_number.valueOf()) + 8"
                @scrollend.prevent="onScrollEnd"
                ref="kyou_list_image_view">
                <template v-slot:default="{ item }">
                    <table>
                        <tr>
                            <td v-for="kyou in item" :key="kyou.id">
                                <KyouView class="kyou_image_in_list" :application_config="application_config"
                                    :draggable="draggable" :key="kyou.id" :gkill_api="gkill_api" :highlight_targets="[]"
                                    :is_image_view="true" :kyou="kyou"
                                    :show_checkbox="false" :show_content_only="true" :show_mi_create_time="true"
                                    :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                                    :show_mi_limit_time="true" :show_timeis_elapsed_time="true"
                                    :show_timeis_plaing_end_button="true" :height="'100%'" :width="'100%'"
                                    :is_readonly_mi_check="true" :enable_context_menu="enable_context_menu"
                                    :enable_dialog="enable_dialog" :show_attached_timeis="false" :show_rep_name="true"
                                    :force_show_latest_kyou_info="false" :show_update_time="false"
                                    :show_related_time="!(query.for_mi && kyou.data_type === 'mi_create' && (query.include_start_mi || query.include_end_mi || query.include_limit_mi))"
                                    :show_attached_tags="false" :show_attached_texts="false"
                                    :show_attached_notifications="false" :is_image_request_to_thumb_size="true"
                                    @focused_kyou="onFocusedKyou"
                                    @clicked_kyou="onClickedKyou"
                                    v-on="crudRelayHandlers" />
                            </td>
                        </tr>
                    </table>
                </template>
            </v-virtual-scroll>
        </v-card>
        <v-card v-if="show_footer" :class="footer_class" variant="text" :ripple="false" :link="false">
            <v-row no-gutters>
                <v-col v-if="matched_kyous && matched_kyous.length" cols="auto" class="py-3">
                    {{ matched_kyous.length }}{{ i18n.global.t("N_COUNT_ITEMS_TITLE") }}
                </v-col>
                <v-spacer />

                <v-col cols="auto" class="pa-0">
                    <v-btn class="rounded-sm mx-auto" icon @click.prevent="onRequestedSearch" variant="text">
                        <v-icon>mdi-reload</v-icon>
                    </v-btn>
                </v-col>

                <v-col cols="auto" class="pa-0" v-if="is_show_doc_image_toggle_button">
                    <v-btn class="rounded-sm mx-auto" icon
                        @click.prevent="onRequestedChangeImageOnly"
                        variant="text">
                        <v-icon v-show="!query.is_image_only">mdi-file-document-outline</v-icon>
                        <v-icon v-show="query.is_image_only">mdi-image</v-icon>
                    </v-btn>
                </v-col>

                <v-col cols="auto" class="pa-0" v-if="is_show_arrow_button">
                    <v-btn class="rounded-sm mx-auto" icon variant="text"
                        @click.prevent="onRequestedChangeFocusKyou">
                        <v-icon v-show="!query.is_focus_kyou_in_list_view">mdi-arrow-down</v-icon>
                        <v-icon v-show="query.is_focus_kyou_in_list_view">mdi-arrow-right</v-icon>
                    </v-btn>
                </v-col>

                <v-col cols="auto" class="pa-0">
                    <v-btn class="rounded-sm mx-auto" icon
                        @click.prevent="onRequestedCloseColumn"
                        :disabled="!closable" variant="text">
                        <v-icon v-show="closable">mdi-close</v-icon>
                    </v-btn>
                </v-col>
            </v-row>
        </v-card>
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { KyouListViewEmits } from './kyou-list-view-emits'
import type { KyouListViewProps } from './kyou-list-view-props'
import KyouView from './kyou-view.vue'
import { useKyouListView } from '@/classes/use-kyou-list-view'

const props = defineProps<KyouListViewProps>()
const emits = defineEmits<KyouListViewEmits>()

const {
    // Template refs
    kyou_list_view,
    kyou_list_image_view,

    // State
    match_kyous_for_image,
    is_loading,

    // Computed
    kyou_height_px,
    footer_height,
    footer_class,

    // Exposed methods
    scroll_to,
    scroll_to_kyou,
    scroll_to_time,
    set_loading,
    get_is_loading,
    get_query_id,

    // Template event handlers
    onScrollEnd,
    onClickedListView,
    onFocusedKyou,
    onClickedKyou,
    onRequestedSearch,
    onRequestedChangeImageOnly,
    onRequestedChangeFocusKyou,
    onRequestedCloseColumn,

    // Event relay objects
    crudRelayHandlers,
} = useKyouListView({ props, emits })

defineExpose({ scroll_to, scroll_to_kyou, scroll_to_time, set_loading, get_is_loading, get_query_id })
</script>

<style lang="css" scoped>
.kyou_in_list {
    overflow-y: hidden !important;
    height: v-bind(kyou_height_px) !important;
    min-height: v-bind(kyou_height_px) !important;
    max-height: v-bind(kyou_height_px) !important;
    border-top: 1px solid silver;
}

.kyou_image_in_list {
    height: 200px;
    width: 200px;
}

.focused_list>* {
    background-color: rgb(var(--v-theme-background-focused));
}
</style>
<style lang="css" scoped>
.kyou_list_view_card_wrap .kyou_list_view_card {
    overflow-y: hidden !important;
}
</style>
