<template>
    <div @dblclick="show_kyou_dialog()">
        <AttachedTag v-for="attached_tag, index in kyou.attached_tags" :tag="attached_tag"
            :application_config="application_config" :gkill_api="gkill_api" :kyou="kyou"
            :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <AttachedTimeIsPlaing v-for="attached_timeis_plaing, index in kyou.attached_timeis_kyou"
            :timeis_kyou="attached_timeis_plaing" :application_config="application_config" :gkill_api="gkill_api"
            :kyou="kyou" :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <kyouDialog :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
            ref="kyou_dialog" />
        <EditKmemoDialog v-if="kyou.typed_kmemo" :kmemo="kyou.typed_kmemo" :application_config="application_config"
            :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <EditMiDialog v-if="kyou.typed_mi" :mi="kyou.typed_mi" :application_config="application_config"
            :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <EditNlogDialog v-if="kyou.typed_nlog" :nlog="kyou.typed_nlog" :application_config="application_config"
            :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <EditLantanaDialog v-if="kyou.typed_lantana" :application_config="application_config"
            :lantana="kyou.typed_lantana" :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="kyou"
            :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <EditTimeIsDialog v-if="kyou.typed_timeis" :timeis="kyou.typed_timeis" :application_config="application_config"
            :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <EditURLogDialog v-if="kyou.typed_urlog" :urlog="kyou.typed_urlog" :application_config="application_config"
            :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <EditIDFKyouDialog v-if="kyou.typed_idf_kyou" :idf_kyou="kyou.typed_idf_kyou"
            :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
            :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <ConfirmDeleteKyouDialog :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <KyouHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <AddTagDialog :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <v-row class="pa-0 ma-0">
            <v-col class="kyou_related_time pa-0 ma-0" cols="auto">
                {{ format_time(kyou.related_time) }}
            </v-col>
            <v-spacer />
            <v-col class="kyou_data_type pa-0 ma-0" cols="auto">
                {{ kyou.data_type }}
            </v-col>
            <v-col class="kyou_rep_name pa-0 ma-0" cols="auto">
                _{{ kyou.rep_name }}_
            </v-col>
            <v-col class="kyou_device pa-0 ma-0" cols="auto">
                {{ kyou.update_device }}
            </v-col>
        </v-row>
        <div :class="kyou_class">
            <KmemoView v-if="kyou.typed_kmemo" :kmemo="kyou.typed_kmemo" :application_config="application_config"
                :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="kyou"
                :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <miKyouView v-if="kyou.typed_mi" :mi="kyou.typed_mi" :application_config="application_config"
                :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="kyou"
                :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <NlogView v-if="kyou.typed_nlog" :nlog="kyou.typed_nlog" :application_config="application_config"
                :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="kyou"
                :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <LantanaView v-if="kyou.typed_lantana" :lantana="kyou.typed_lantana"
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="kyou" :last_added_tag="last_added_tag"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <TimeIsView v-if="kyou.typed_timeis" :timeis="kyou.typed_timeis" :show_timeis_plaing_end_button="true"
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="kyou" :last_added_tag="last_added_tag"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <URLogView v-if="kyou.typed_urlog" :urlog="kyou.typed_urlog" :application_config="application_config"
                :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="kyou"
                :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <IDFKyouView v-if="kyou.typed_idf_kyou" :idf_kyou="kyou.typed_idf_kyou"
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="kyou" :last_added_tag="last_added_tag"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <ReKyouView v-if="kyou.typed_rekyou" :rekyou="kyou.typed_rekyou" :application_config="application_config"
                :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="kyou"
                :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <GitCommitLogView v-if="kyou.typed_git_commit_log" :git_commit_log="kyou.typed_git_commit_log"
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="kyou" :last_added_tag="last_added_tag"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </div>
        <AttachedText v-for="attached_text, index in kyou.attached_texts" :text="attached_text"
            :application_config="application_config" :gkill_api="gkill_api" :kyou="kyou"
            :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    </div>
</template>
<script setup lang="ts">
import { computed, type Ref, ref } from 'vue'

import AddTagDialog from '../dialogs/add-tag-dialog.vue'
import AddTextDialog from '../dialogs/add-text-dialog.vue'
import AttachedTag from './attached-tag.vue'
import AttachedText from './attached-text.vue'
import AttachedTimeIsPlaing from './attached-time-is-plaing.vue'
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue'
import ConfirmDeleteTagDialog from '../dialogs/confirm-delete-tag-dialog.vue'
import ConfirmDeleteTextDialog from '../dialogs/confirm-delete-text-dialog.vue'
import EditIDFKyouDialog from '../dialogs/edit-idf-kyou-dialog.vue'
import EditKmemoDialog from '../dialogs/edit-kmemo-dialog.vue'
import EditLantanaDialog from '../dialogs/edit-lantana-dialog.vue'
import EditMiDialog from '../dialogs/edit-mi-dialog.vue'
import EditNlogDialog from '../dialogs/edit-nlog-dialog.vue'
import EditTagDialog from '../dialogs/edit-tag-dialog.vue'
import EditTextDialog from '../dialogs/edit-text-dialog.vue'
import EditTimeIsDialog from '../dialogs/edit-time-is-dialog.vue'
import EditURLogDialog from '../dialogs/edit-ur-log-dialog.vue'
import GitCommitLogView from './git-commit-log-view.vue'
import IDFKyouView from './idf-kyou-view.vue'
import KmemoView from './kmemo-view.vue'
import KyouHistoriesDialog from '../dialogs/kyou-histories-dialog.vue'
import LantanaView from './lantana-view.vue'
import miKyouView from './mi-kyou-view.vue'
import NlogView from './nlog-view.vue'
import ReKyouView from './re-kyou-view.vue'
import TagHistoriesDialog from '../dialogs/tag-histories-dialog.vue'
import TextHistoriesDialog from '../dialogs/text-histories-dialog.vue'
import TimeIsView from './time-is-view.vue'
import URLogView from './ur-log-view.vue'
import kyouDialog from '../dialogs/kyou-dialog.vue'

import type { KyouViewEmits } from './kyou-view-emits'
import type { KyouViewProps } from './kyou-view-props'
import { Tag } from '@/classes/datas/tag'
import { Text } from '@/classes/datas/text'

const kyou_dialog = ref<InstanceType<typeof kyouDialog> | null>(null);

const props = defineProps<KyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const focused_tag: Ref<Tag> = ref(new Tag())
const focused_text: Ref<Text> = ref(new Text())
const delete_target_tag: Ref<Tag> = ref(new Tag())
const delete_target_text: Ref<Text> = ref(new Text())

const kyou_class = computed(() => {
    let highlighted = false;
    for (let i = 0; i < props.highlight_targets.length; i++) {
        if (props.highlight_targets[i].id === props.kyou.id
            && props.highlight_targets[i].create_time === props.kyou.create_time
            && props.highlight_targets[i].update_time === props.kyou.update_time) {
            highlighted = true
            break
        }
    }
    if (highlighted) {
        return "highlighted_kyou"
    }
    return "kyou"
})

async function load_attached_infos(): Promise<void> {
    throw new Error('Not implemented')
}
async function clear_attached_infos(): Promise<void> {
    throw new Error('Not implemented')
}
async function update_check(is_checked: boolean): Promise<void> {
    throw new Error('Not implemented')
}

function show_kyou_dialog(): void {
    kyou_dialog.value?.show()
}

function format_time(time: Date) {
    let year: string | number = time.getFullYear()
    let month: string | number = time.getMonth() + 1
    let date: string | number = time.getDate()
    let hour: string | number = time.getHours()
    let minute: string | number = time.getMinutes()
    let second: string | number = time.getSeconds()
    const day_of_week = ['日', '月', '火', '水', '木', '金', '土'][time.getDay()]
    month = ('0' + month).slice(-2)
    date = ('0' + date).slice(-2)
    hour = ('0' + hour).slice(-2)
    minute = ('0' + minute).slice(-2)
    second = ('0' + second).slice(-2)
    return year + '/' + month + '/' + date + '(' + day_of_week + ')' + ' ' + hour + ':' + minute + ':' + second
}

</script>
<style lang="css">
.kyou .v-card,
.highlighted_kyou .v-card {
    background-color: #776ef300;
}
</style>
<style lang="css" scoped>
.highlighted_kyou {
    background: lightgreen;
}

.kyou {
    background-color: #776ef300;
}

.kyou_related_time,
.kyou_rep_name,
.kyou_device,
.kyou_data_type {
    font-size: small;
    color: silver;
}
</style>
