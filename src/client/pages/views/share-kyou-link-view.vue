<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("SHARED_KYOU_LINK_LIST_TITLE") }}
        </v-card-title>
        <v-text-field v-model="share_title" :label="i18n.global.t('SHARE_KYOU_TITLE_TITLE')" />
        <v-select v-model="view_type" :items="view_types" :label="i18n.global.t('SHARE_KYOU_VIEW_TYPE')" />
        <v-checkbox v-model="is_share_time_only" :label="i18n.global.t('SHARE_KYOU_SHARE_TIME_ONLY')" />
        <v-checkbox v-model="is_share_with_tags" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_TAGS')" />
        <v-checkbox v-model="is_share_with_texts" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_TEXTS')" />
        <v-checkbox v-model="is_share_with_timeiss" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_TIMEISS')" />
        <v-checkbox v-model="is_share_with_locations" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_LOCATIONS')" />
        <v-text-field v-model="local_share_url" :label="i18n.global.t('LOCAL_TITLE')" readonly
            @click="copy_local_share_kyou_link" @focus="$event.target.select()" />
        <v-text-field v-model="lan_share_url" :label="i18n.global.t('IN_LAN_TITLE')" readonly
            @click="copy_lan_share_kyou_link" @focus="$event.target.select()" />
        <v-text-field v-model="over_lan_share_url" :label="i18n.global.t('OVER_LAN_TITLE')" readonly
            @click="copy_over_lan_share_kyou_link" @focus="$event.target.select()" />
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col class="pa-0 ma-0" cols="auto">
                <v-btn dark color="primary" @click="update()">{{ i18n.global.t("UPDATE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ShareKyousLinkViewEmits } from './share-kyou-link-view-emits'
import type { ShareKyousLinkViewProps } from './share-kyou-link-view-props'
import { useShareKyouLinkView } from '@/classes/use-share-kyou-link-view'

const props = defineProps<ShareKyousLinkViewProps>()
const emits = defineEmits<ShareKyousLinkViewEmits>()

const {
    // State
    share_title,
    view_type,
    view_types,
    is_share_time_only,
    is_share_with_tags,
    is_share_with_texts,
    is_share_with_timeiss,
    is_share_with_locations,
    local_share_url,
    lan_share_url,
    over_lan_share_url,

    // Business logic
    copy_local_share_kyou_link,
    copy_lan_share_kyou_link,
    copy_over_lan_share_kyou_link,
    update,
} = useShareKyouLinkView({ props, emits })
</script>
