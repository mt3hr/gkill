<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("DELETE_KYOU_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="delete_kyou()">{{ i18n.global.t("DELETE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :is_readonly_mi_check="false" :show_attached_timeis="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_related_time="true"
                @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
                @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
                @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
                @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
                @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
                @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
                @registered_text="(registered_text) => emits('registered_text', registered_text)"
                @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
                @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
                @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
                @updated_text="(updated_text) => emits('updated_text', updated_text)"
                @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import type { KyouViewEmits } from './kyou-view-emits'
import { type Ref, ref, watch } from 'vue'
import KyouView from './kyou-view.vue'
import type { Kyou } from '@/classes/datas/kyou';
import type { GkillError } from '@/classes/api/gkill-error';
import { UpdateKmemoRequest } from '@/classes/api/req_res/update-kmemo-request';
import { UpdateKCRequest } from '@/classes/api/req_res/update-kc-request';
import { UpdateURLogRequest } from '@/classes/api/req_res/update-ur-log-request';
import { UpdateNlogRequest } from '@/classes/api/req_res/update-nlog-request';
import { UpdateTimeisRequest } from '@/classes/api/req_res/update-timeis-request';
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request';
import { UpdateLantanaRequest } from '@/classes/api/req_res/update-lantana-request';
import { UpdateReKyouRequest } from '@/classes/api/req_res/update-re-kyou-request';
import type { ConfirmDeleteKyouViewProps } from './confirm-delete-kyou-view-props';
import { UpdateIDFKyouRequest } from '@/classes/api/req_res/update-idf-kyou-request';
import delete_gkill_cache from '@/classes/delete-gkill-cache';

const props = defineProps<ConfirmDeleteKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
watch(() => props.kyou, () => cloned_kyou.value = props.kyou.clone())

const show_kyou: Ref<boolean> = ref(true)

async function delete_kyou(): Promise<void> {
    let errors = new Array<GkillError>()
    await cloned_kyou.value.load_typed_datas()
    if (cloned_kyou.value.data_type.startsWith("kmemo")) {
        const e = await delete_kmemo()
        errors = errors.concat(e)
    }
    if (cloned_kyou.value.data_type.startsWith("kc")) {
        const e = await delete_kc()
        errors = errors.concat(e)
    }
    if (cloned_kyou.value.data_type.startsWith("urlog")) {
        const e = await delete_urlog()
        errors = errors.concat(e)
    }
    if (cloned_kyou.value.data_type.startsWith("nlog")) {
        const e = await delete_nlog()
        errors = errors.concat(e)
    }
    if (cloned_kyou.value.data_type.startsWith("timeis")) {
        const e = await delete_timeis()
        errors = errors.concat(e)
    }
    if (cloned_kyou.value.data_type.startsWith("mi")) {
        const e = await delete_mi()
        errors = errors.concat(e)
    }
    if (cloned_kyou.value.data_type.startsWith("lantana")) {
        const e = await delete_lantana()
        errors = errors.concat(e)
    }
    if (cloned_kyou.value.data_type.startsWith("idf")) {
        const e = await delete_idf_kyou()
        errors = errors.concat(e)
    }
    if (cloned_kyou.value.data_type.startsWith("git")) {
        const e = await delete_git_commit_log()
        errors = errors.concat(e)
    }
    if (cloned_kyou.value.data_type.startsWith("rekyou")) {
        const e = await delete_rekyou()
        errors = errors.concat(e)
    }
    if (errors && errors.length != 0) {
        emits('received_errors', errors)
    }
    emits('requested_reload_list')
    emits('requested_close_dialog')
}

async function delete_kmemo(): Promise<Array<GkillError>> {
    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)

    await delete_gkill_cache(cloned_kyou.value.id)
    const req = new UpdateKmemoRequest()
    req.kmemo = cloned_kyou.value.typed_kmemo!!.clone()
    req.kmemo.is_deleted = true

    req.kmemo.update_app = "gkill"
    req.kmemo.update_device = gkill_info_res.device
    req.kmemo.update_time = new Date(Date.now())
    req.kmemo.update_user = gkill_info_res.user_id

    const res = await props.gkill_api.update_kmemo(req)
    return res.errors
}
async function delete_kc(): Promise<Array<GkillError>> {
    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)

    await delete_gkill_cache(cloned_kyou.value.id)
    const req = new UpdateKCRequest()
    req.kc = cloned_kyou.value.typed_kc!!.clone()
    req.kc.is_deleted = true

    req.kc.update_app = "gkill"
    req.kc.update_device = gkill_info_res.device
    req.kc.update_time = new Date(Date.now())
    req.kc.update_user = gkill_info_res.user_id

    const res = await props.gkill_api.update_kc(req)
    return res.errors
}
async function delete_urlog(): Promise<Array<GkillError>> {
    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)

    await delete_gkill_cache(cloned_kyou.value.id)
    const req = new UpdateURLogRequest()
    req.urlog = cloned_kyou.value.typed_urlog!!.clone()
    req.urlog.is_deleted = true

    req.urlog.update_app = "gkill"
    req.urlog.update_device = gkill_info_res.device
    req.urlog.update_time = new Date(Date.now())
    req.urlog.update_user = gkill_info_res.user_id

    const res = await props.gkill_api.update_urlog(req)
    return res.errors
}

async function delete_nlog(): Promise<Array<GkillError>> {
    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)

    await delete_gkill_cache(cloned_kyou.value.id)
    const req = new UpdateNlogRequest()
    req.nlog = cloned_kyou.value.typed_nlog!!.clone()
    req.nlog.is_deleted = true

    req.nlog.update_app = "gkill"
    req.nlog.update_device = gkill_info_res.device
    req.nlog.update_time = new Date(Date.now())
    req.nlog.update_user = gkill_info_res.user_id

    const res = await props.gkill_api.update_nlog(req)
    return res.errors
}

async function delete_timeis(): Promise<Array<GkillError>> {
    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)

    await delete_gkill_cache(cloned_kyou.value.id)
    const req = new UpdateTimeisRequest()
    req.timeis = cloned_kyou.value.typed_timeis!!.clone()
    req.timeis.is_deleted = true

    req.timeis.update_app = "gkill"
    req.timeis.update_device = gkill_info_res.device
    req.timeis.update_time = new Date(Date.now())
    req.timeis.update_user = gkill_info_res.user_id

    const res = await props.gkill_api.update_timeis(req)
    return res.errors
}

async function delete_mi(): Promise<Array<GkillError>> {
    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)

    await delete_gkill_cache(cloned_kyou.value.id)
    const req = new UpdateMiRequest()
    req.mi = cloned_kyou.value.typed_mi!!.clone()
    req.mi.is_deleted = true

    req.mi.update_app = "gkill"
    req.mi.update_device = gkill_info_res.device
    req.mi.update_time = new Date(Date.now())
    req.mi.update_user = gkill_info_res.user_id

    const res = await props.gkill_api.update_mi(req)
    return res.errors
}

async function delete_lantana(): Promise<Array<GkillError>> {
    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)

    await delete_gkill_cache(cloned_kyou.value.id)
    const req = new UpdateLantanaRequest()
    req.lantana = cloned_kyou.value.typed_lantana!!.clone()
    req.lantana.is_deleted = true

    req.lantana.update_app = "gkill"
    req.lantana.update_device = gkill_info_res.device
    req.lantana.update_time = new Date(Date.now())
    req.lantana.update_user = gkill_info_res.user_id


    const res = await props.gkill_api.update_lantana(req)
    return res.errors
}

async function delete_idf_kyou(): Promise<Array<GkillError>> {
    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)

    await delete_gkill_cache(cloned_kyou.value.id)
    const req = new UpdateIDFKyouRequest()
    req.idf_kyou = cloned_kyou.value.typed_idf_kyou!!.clone()
    req.idf_kyou.is_deleted = true

    req.idf_kyou.update_app = "gkill"
    req.idf_kyou.update_device = gkill_info_res.device
    req.idf_kyou.update_time = new Date(Date.now())
    req.idf_kyou.update_user = gkill_info_res.user_id

    const res = await props.gkill_api.update_idf_kyou(req)
    return res.errors
}

async function delete_git_commit_log(): Promise<Array<GkillError>> {
    throw new Error("not implements")
}

async function delete_rekyou(): Promise<Array<GkillError>> {
    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)

    await delete_gkill_cache(cloned_kyou.value.id)
    const req = new UpdateReKyouRequest()
    req.rekyou = cloned_kyou.value.typed_rekyou!!.clone()
    req.rekyou.is_deleted = true

    req.rekyou.update_app = "gkill"
    req.rekyou.update_device = gkill_info_res.device
    req.rekyou.update_time = new Date(Date.now())
    req.rekyou.update_user = gkill_info_res.user_id

    const res = await props.gkill_api.update_rekyou(req)
    return res.errors
}
</script>
