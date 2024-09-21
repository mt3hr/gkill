<template>
    <!--//TODO 実装-->
    <EditTagDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_tag(cloned_tag)]" :kyou="cloned_kyou"
        :last_added_tag="last_added_tag" :tag="cloned_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <ConfirmDeleteTagDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_tag(cloned_tag)]" :kyou="cloned_kyou"
        :last_added_tag="last_added_tag" :tag="cloned_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <TagHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_tag(cloned_tag)]" :kyou="cloned_kyou"
        :last_added_tag="last_added_tag" :tag="cloned_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script lang="ts" setup>
import type { AttachedTagContextMenuProps } from './attached-tag-context-menu-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { Tag } from '@/classes/datas/tag';
import { type Ref, ref } from 'vue';
import EditTagDialog from '../dialogs/edit-tag-dialog.vue';
import ConfirmDeleteTagDialog from '../dialogs/confirm-delete-tag-dialog.vue';
import TagHistoriesDialog from '../dialogs/tag-histories-dialog.vue';
import { InfoIdentifier } from '@/classes/datas/info-identifier';
import type { Kyou } from '@/classes/datas/kyou';

const props = defineProps<AttachedTagContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_tag: Ref<Tag> = ref(await props.tag.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());

function generate_info_identifer_from_tag(tag: Tag): InfoIdentifier {
    const info_identifer = new InfoIdentifier()
    info_identifer.create_time = tag.create_time
    info_identifer.id = tag.id
    info_identifer.update_time = tag.update_time
    return info_identifer
}
</script>
