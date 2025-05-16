<template>
    <div>
        <div class="overlay_target">
            <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                <v-progress-circular indeterminate color="primary" />
            </v-overlay>
        </div>
        <SharedMiPage
            v-if="!is_loading && view_type === 'mi' && application_config && gkill_api_for_share && share_title"
            :gkill_api="gkill_api_for_share" :application_config="application_config" :share_title="share_title"
            :share_id="share_id" />
        <SharedRYKVPage
            v-if="!is_loading && view_type === 'rykv' && application_config && gkill_api_for_share && share_title"
            :gkill_api="gkill_api_for_share" :application_config="application_config" :share_title="share_title"
            :share_id="share_id" />
    </div>
</template>

<script lang="ts" setup>
'use strict'

import { GkillAPI, GkillAPIForSharedKyou } from '@/classes/api/gkill-api';
import { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';
import { GetSharedKyousRequest } from '@/classes/api/req_res/get-shared-kyous-request';
import { i18n } from '@/i18n';
import { computed, nextTick, type Ref, ref } from 'vue';
import SharedMiPage from './shared-mi-page.vue';
import SharedRYKVPage from './shared-rykv-page.vue';
import { useRoute } from 'vue-router';
import type { ApplicationConfig } from '@/classes/datas/config/application-config';
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request';

const route = useRoute()
const share_id = computed(() => route.query.share_id!.toString())
const view_type: Ref<string | null> = ref(null)
const share_title: Ref<string | null> = ref(null)
const gkill_api_plane: Ref<GkillAPI | null> = ref(null)
const gkill_api_for_share: Ref<GkillAPI | null> = ref(null)
const application_config: Ref<ApplicationConfig | null> = ref(null)
const is_loading: Ref<boolean> = ref(true)

gkill_api_plane.value = GkillAPI.get_instance()

nextTick(async () => await load_gkill_api_and_application_config())

async function load_gkill_api_and_application_config(): Promise<void> {
    if (!gkill_api_plane.value) {
        return
    }
    try {
        const req = new GetSharedKyousRequest()
        req.shared_id = share_id.value
        const res = await gkill_api_plane.value.get_shared_kyous(req)
        if (res.errors && res.errors.length !== 0) {
            write_errors(res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            write_messages(res.messages)
        }

        // GkillAPIForSharedKyouを設定ここから
        const gkill_api_for_shared_kyou = GkillAPIForSharedKyou.get_instance_for_share_kyou()
        gkill_api_for_shared_kyou.kyous = res.kyous
        gkill_api_for_shared_kyou.kmemos = res.kmemos
        gkill_api_for_shared_kyou.kcs = res.kcs
        gkill_api_for_shared_kyou.mis = res.mis
        gkill_api_for_shared_kyou.nlogs = res.nlogs
        gkill_api_for_shared_kyou.lantanas = res.lantanas
        gkill_api_for_shared_kyou.urlogs = res.urlogs
        gkill_api_for_shared_kyou.idf_kyous = res.idf_kyous
        gkill_api_for_shared_kyou.rekyous = res.rekyous
        gkill_api_for_shared_kyou.git_commit_logs = res.git_commit_logs
        gkill_api_for_shared_kyou.gps_logs = res.gps_logs
        gkill_api_for_shared_kyou.attached_tags = res.attached_tags
        gkill_api_for_shared_kyou.attached_texts = res.attached_texts
        gkill_api_for_shared_kyou.attached_timeiss = res.attached_timeiss
        GkillAPI.set_gkill_api(gkill_api_for_shared_kyou)
        // GkillAPIForSharedKyouを設定ここまで

        gkill_api_for_share.value = gkill_api_for_shared_kyou
        gkill_api_for_shared_kyou.set_shared_id_to_cookie(share_id.value)
        application_config.value = (await gkill_api_for_share.value.get_application_config(new GetApplicationConfigRequest())).application_config
        share_title.value = res.title
        view_type.value = res.view_type
        is_loading.value = false
    } catch (e) {
        console.error(e)
        const error = new GkillError()
        error.error_code = GkillErrorCodes.failed_shared_kyous
        error.error_message = i18n.global.t("FAILED_LOAD_MESSAGE")
        write_errors([error])
    }
}

const messages: Ref<Array<{ message: string, id: string, show_snackbar: boolean }>> = ref([])

async function write_errors(errors: Array<GkillError>) {
    const received_messages = new Array<{ message: string, id: string, show_snackbar: boolean }>()
    for (let i = 0; i < errors.length; i++) {
        if (errors[i] && errors[i].error_message) {
            received_messages.push({
                message: errors[i].error_message,
                id: GkillAPI.get_gkill_api().generate_uuid(),
                show_snackbar: true,
            })
        }
    }
    messages.value.push(...received_messages)
    sleep(2500).then(() => {
        for (let i = 0; i < received_messages.length; i++) {
            messages.value.splice(0, 1)
        }
    })
}

async function write_messages(messages_: Array<GkillMessage>) {
    const received_messages = new Array<{ message: string, id: string, show_snackbar: boolean }>()
    for (let i = 0; i < messages_.length; i++) {
        if (messages_[i] && messages_[i].message) {
            received_messages.push({
                message: messages_[i].message,
                id: GkillAPI.get_gkill_api().generate_uuid(),
                show_snackbar: true,
            })
        }
    }
    messages.value.push(...received_messages)
    sleep(2500).then(() => {
        for (let i = 0; i < received_messages.length; i++) {
            messages.value.splice(0, 1)
        }
    })
}
const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

</script>

<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(100vh);
    min-width: calc(100vw);
}
</style>