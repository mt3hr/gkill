<template>
    <div @dblclick="show_kyou_dialog()" @click="emits('clicked_kyou', cloned_kyou)">
        <div v-if="!show_content_only">
            <AttachedTag v-for="attached_tag, index in cloned_kyou.attached_tags" :tag="attached_tag"
                :application_config="application_config" :gkill_api="gkill_api" :kyou="cloned_kyou"
                :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(cloned_kyou) => emits('requested_reload_kyou', cloned_kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(cloned_kyous, is_checked) => emits('requested_update_check_kyous', cloned_kyous, is_checked)" />
            <AttachedTimeIsPlaing v-for="attached_timeis_plaing, index in cloned_kyou.attached_timeis_kyou"
                :timeis_kyou="attached_timeis_plaing" :application_config="application_config" :gkill_api="gkill_api"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(cloned_kyou) => emits('requested_reload_kyou', cloned_kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(cloned_kyous, is_checked) => emits('requested_update_check_kyous', cloned_kyous, is_checked)" />
            <v-row class="pa-0 ma-0">
                <v-col class="kyou_related_time pa-0 ma-0" cols="auto">
                    {{ format_time(cloned_kyou.related_time) }}
                </v-col>
                <v-spacer />
                <v-col class="kyou_data_type pa-0 ma-0" cols="auto">
                    {{ cloned_kyou.data_type }}
                </v-col>
                <v-col class="kyou_rep_name pa-0 ma-0" cols="auto">
                    _{{ cloned_kyou.rep_name }}_
                </v-col>
                <v-col class="kyou_device pa-0 ma-0" cols="auto">
                    {{ cloned_kyou.update_device }}
                </v-col>
            </v-row>
        </div>
        <div :class="kyou_class">
            <KmemoView v-if="cloned_kyou.typed_kmemo" :kmemo="cloned_kyou.typed_kmemo"
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <miKyouView v-if="cloned_kyou.typed_mi" :mi="cloned_kyou.typed_mi" :application_config="application_config"
                :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="cloned_kyou"
                :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
                :height="height" :width="width" :is_readonly_mi_check="is_readonly_mi_check"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <NlogView v-if="cloned_kyou.typed_nlog" :nlog="cloned_kyou.typed_nlog"
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <LantanaView v-if="cloned_kyou.typed_lantana" :lantana="cloned_kyou.typed_lantana"
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <TimeIsView v-if="cloned_kyou.typed_timeis" :timeis="cloned_kyou.typed_timeis"
                :show_timeis_plaing_end_button="true" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :kyou="cloned_kyou" :last_added_tag="last_added_tag"
                :height="height" :width="width" @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <URLogView v-if="cloned_kyou.typed_urlog" :urlog="cloned_kyou.typed_urlog"
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <IDFKyouView v-if="cloned_kyou.typed_idf_kyou" :idf_kyou="cloned_kyou.typed_idf_kyou"
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <ReKyouView v-if="cloned_kyou.typed_rekyou" :rekyou="cloned_kyou.typed_rekyou"
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            <GitCommitLogView v-if="cloned_kyou.typed_git_commit_log" :git_commit_log="cloned_kyou.typed_git_commit_log"
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </div>
        <div v-if="!show_content_only">
            <AttachedText v-for="attached_text, index in cloned_kyou.attached_texts" :text="attached_text"
                :application_config="application_config" :gkill_api="gkill_api" :kyou="cloned_kyou"
                :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', cloned_kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </div>
        <kyouDialog :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="cloned_kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(cloned_kyou) => emits('requested_reload_kyou', cloned_kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(cloned_kyous, is_checked) => emits('requested_update_check_kyous', cloned_kyous, is_checked)"
            ref="kyou_dialog" />
    </div>
</template>
<script setup lang="ts">
import { computed, watch, type Ref, ref, onMounted, onUnmounted } from 'vue'

import AttachedTag from './attached-tag.vue'
import AttachedText from './attached-text.vue'
import AttachedTimeIsPlaing from './attached-time-is-plaing.vue'
import GitCommitLogView from './git-commit-log-view.vue'
import IDFKyouView from './idf-kyou-view.vue'
import KmemoView from './kmemo-view.vue'
import LantanaView from './lantana-view.vue'
import miKyouView from './mi-kyou-view.vue'
import NlogView from './nlog-view.vue'
import ReKyouView from './re-kyou-view.vue'
import TimeIsView from './time-is-view.vue'
import URLogView from './ur-log-view.vue'
import kyouDialog from '../dialogs/kyou-dialog.vue'

import type { KyouViewEmits } from './kyou-view-emits'
import type { KyouViewProps } from './kyou-view-props'
import { Kyou } from '@/classes/datas/kyou'
import { Tag } from '@/classes/datas/tag'
import { Text } from '@/classes/datas/text'

const kyou_dialog = ref<InstanceType<typeof kyouDialog> | null>(null);

const props = defineProps<KyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
const focused_tag: Ref<Tag> = ref(new Tag())
const focused_text: Ref<Text> = ref(new Text())
const delete_target_tag: Ref<Tag> = ref(new Tag())
const delete_target_text: Ref<Text> = ref(new Text())

watch(() => props.kyou, async () => {
    cloned_kyou.value = props.kyou.clone()
    return load_attached_infos()
})

onMounted(() => {
    load_attached_infos()
})

onUnmounted(() => {
    clear_attached_infos()
})

load_attached_infos()

const kyou_class = computed(() => {
    let highlighted = false;
    for (let i = 0; i < props.highlight_targets.length; i++) {
        if (props.highlight_targets[i].id === props.kyou.id
            && props.highlight_targets[i].create_time.getTime() === props.kyou.create_time.getTime()
            && props.highlight_targets[i].update_time.getTime() === props.kyou.update_time.getTime()) {
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
    if (cloned_kyou.value.abort_controller) {
        cloned_kyou.value.abort_controller.abort()
    }
    cloned_kyou.value.abort_controller = new AbortController()

    try {
        const errors = await cloned_kyou.value.load_all()

        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
        }
    } catch (err: any) {
        // abortは握りつぶす
        if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
            // abort以外はエラー出力する
            console.error(err)
        }
    }
}
async function clear_attached_infos(): Promise<void> {
    if (cloned_kyou.value.abort_controller) {
        cloned_kyou.value.abort_controller.abort()
    }
    cloned_kyou.value.abort_controller = new AbortController()

    try {
        const errors = await cloned_kyou.value.clear_all()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
        }
    } catch (err: any) {
        // abortは握りつぶす
        if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
            // abort以外はエラー出力する
            console.error(err)
        }
    }
}
async function update_check(is_checked: boolean): Promise<void> {
    emits('requested_update_check_kyous', [cloned_kyou.value], is_checked)
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
<style lang="css" scoped>
.highlighted_kyou>* {
    background-color: lightgreen;
}

.kyou_related_time,
.kyou_rep_name,
.kyou_device,
.kyou_data_type {
    font-size: small;
    color: silver;
}
</style>
