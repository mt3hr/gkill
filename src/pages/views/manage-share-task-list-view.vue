<template>
    <v-card>
        <v-card-title>
            {{ $t("SHARE_KYOU_LINK_TITLE") }}
        </v-card-title>
        <table>
            <tr v-for="share_kyou_list_info in share_kyou_list_infos" :key="share_kyou_list_info.share_id">
                <td>
                    {{ share_kyou_list_info.share_title }}
                </td>
                <td>
                    <v-btn color="primary"
                        @click="emits('requested_show_share_kyou_link_dialog', share_kyou_list_info)">
                        {{ $t("LINK_TITLE") }}
                    </v-btn>
                </td>
                <td>
                    <v-btn dark color="secondary"
                        @click="emits('requested_show_confirm_delete_share_kyou_list_dialog', share_kyou_list_info)">
                        {{ $t("DELETE_TITLE") }}
                    </v-btn>
                </td>
            </tr>
        </table>
    </v-card>
</template>
<script lang="ts" setup>
import type { ManageShareKyousListViewEmits } from './manage-share-task-list-view-emits'
import type { ManageShareKyousListViewProps } from './manage-share-task-list-view-props'

defineProps<ManageShareKyousListViewProps>()
const emits = defineEmits<ManageShareKyousListViewEmits>()
</script>
