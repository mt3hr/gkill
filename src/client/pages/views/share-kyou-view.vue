<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("SHARE_KYOU_TITLE") }}
        </v-card-title>
        <div>{{ i18n.global.t("SHARE_KYOU_MESSAGE") }}</div>
        <v-text-field v-model="share_title" :label="i18n.global.t('SHARE_KYOU_TITLE_TITLE')" item-title="title"
            item-value="value" />
        <v-select v-model="view_type" :items="view_types" :label="i18n.global.t('SHARE_KYOU_VIEW_TYPE')" />
        <v-checkbox v-model="is_share_time_only" :label="i18n.global.t('SHARE_KYOU_SHARE_TIME_ONLY')" />
        <v-checkbox v-model="is_share_with_tags" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_TAGS')" />
        <v-checkbox v-model="is_share_with_texts" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_TEXTS')" />
        <v-checkbox v-model="is_share_with_timeiss" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_TIMEISS')" />
        <v-checkbox v-model="is_share_with_locations" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_LOCATIONS')" />
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col class="pa-0 ma-0" cols="auto">
                <v-btn dark color="primary" @click="share()">{{ i18n.global.t("OK_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ShareKyousListViewEmits } from './share-kyou-view-emits'
import type { ShareKyousListViewProps } from './share-kyou-view-props'
import { useShareKyouView } from '@/classes/use-share-kyou-view'

const props = defineProps<ShareKyousListViewProps>()
const emits = defineEmits<ShareKyousListViewEmits>()

const {
    view_types,
    share_title,
    view_type,
    is_share_time_only,
    is_share_with_tags,
    is_share_with_texts,
    is_share_with_timeiss,
    is_share_with_locations,
    share,
} = useShareKyouView({ props, emits })
</script>
