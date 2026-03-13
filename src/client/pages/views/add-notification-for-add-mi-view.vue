<template>
    <v-card class="pa-0 ma-0">
        <v-textarea v-model="content_value" :label="i18n.global.t('NOTIFICATION_CONTENT_TITLE')" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_notification_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="notification_date_string"
                                        :label="i18n.global.t('NOTIFICATION_DATE_TITLE')" readonly v-bind="props"
                                        min-width="120" />
                                </template>
                                <v-date-picker v-model="notification_date_typed"
                                    @update:model-value="show_notification_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_notification_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="notification_time_string"
                                        :label="i18n.global.t('NOTIFICATION_TIME_TITLE')" readonly min-width="120"
                                        v-bind="props" />
                                </template>
                                <v-time-picker v-model="notification_time_string" format="24hr"
                                    @update:minute="show_notification_time_menu = false" />
                            </v-menu>
                        </td>
                        <td>
                            <v-btn dark color="secondary" @click="reset_notification_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                    </tr>
                </table>
                <v-btn dark color="secondary" @click="reset_notification_date_time()">{{ i18n.global.t("RESET_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { KyouViewEmits } from './kyou-view-emits'
import type { AddNotificationForAddMiViewProps } from './add-notification-for-add-mi-view-props'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import { useAddNotificationForAddMiView } from '@/classes/use-add-notification-for-add-mi-view'

const props = defineProps<AddNotificationForAddMiViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // State
    is_requested_submit,
    content_value,
    notification_date_typed,
    notification_date_string,
    notification_time_string,
    show_notification_date_menu,
    show_notification_time_menu,

    // Business logic
    get_notification,
    reset_notification_date_time,
} = useAddNotificationForAddMiView({ props, emits })

defineExpose({ get_notification })
</script>
