<template>
    <v-card>
        <v-card-title>
            タスク共有リンク一覧
        </v-card-title>
        <table>
            <tr v-for="share_mi_task_list_info in share_mi_task_list_infos" :key="share_mi_task_list_info.share_id">
                <td>
                    {{ share_mi_task_list_info.share_title }}
                </td>
                <td>
                    <v-btn color="primary"
                        @click="emits('requested_show_share_task_link_dialog', share_mi_task_list_info)">
                        リンク
                    </v-btn>
                </td>
                <td>
                    <v-btn color="primary"
                        @click="emits('requested_show_confirm_delete_share_task_list_dialog', share_mi_task_list_info)">
                        削除
                    </v-btn>
                </td>
            </tr>
        </table>
    </v-card>
</template>
<script lang="ts" setup>
import type { ManageShareTaskListViewEmits } from './manage-share-task-list-view-emits'
import type { ManageShareTaskListViewProps } from './manage-share-task-list-view-props'

defineProps<ManageShareTaskListViewProps>()
const emits = defineEmits<ManageShareTaskListViewEmits>()
</script>
