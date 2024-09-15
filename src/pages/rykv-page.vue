<template>
    <v-app-bar class="app_bar" app color="indigo" flat dark :height="app_title_bar_height_px">
        <v-toolbar-title>mi</v-toolbar-title>
        <v-spacer />
    </v-app-bar>
    <v-main class="main">
        <rykvView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api" @received_errors="write_errors"
            @reveived_messages="write_messages" />
        <ApplicationConfigDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :is_show="is_show_application_config_dialog" @received_errors="write_errors"
            @reveived_messages="write_messages" @requested_reload_application_config="load_application_config" />
        <UploadFileDialog />
    </v-main>
</template>

<script lang="ts" setup>
'use strict';
import { computed, ref, type Ref } from 'vue';
import { ApplicationConfig } from '@/classes/datas/config/application-config';
import { GkillAPI } from '@/classes/api/gkill-api';
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

import ApplicationConfigDialog from './dialogs/application-config-dialog.vue';
import UploadFileDialog from './dialogs/upload-file-dialog.vue';
import rykvView from './views/rykv-view.vue';

const actual_height: Ref<Number> = ref(0);
const element_height: Ref<Number> = ref(0);
const browser_url_bar_height: Ref<Number> = ref(0);
const app_title_bar_height: Ref<Number> = ref(50);
const app_title_bar_height_px = computed(() => app_title_bar_height.value.toString().concat("px"))
const gkill_api: Ref<GkillAPI> = ref(new GkillAPI());
const application_config = ref(new ApplicationConfig());
const app_content_height: Ref<Number> = ref(0);
const app_content_width: Ref<Number> = ref(0);

const is_show_application_config_dialog: Ref<boolean> = ref(false);

async function load_application_config(): Promise<void> {
    const req = new GetApplicationConfigRequest()
    req.session_id = ""//TODO session_idをどこから取得するか。webstorage?

    return gkill_api.value.get_application_config(req)
        .then(res => {
            if (res.errors && res.errors.length != 0) {
                write_errors(res.errors)
                return
            }

            application_config.value = res.application_config

            if (res.messages && res.messages.length != 0) {
                write_messages(res.messages)
                return
            }
        })
}

async function resize_content(): Promise<void> {
    const inner_element = document.querySelector('#control-height')
    actual_height.value = window.innerHeight
    element_height.value = inner_element ? inner_element.clientHeight : actual_height.value
    browser_url_bar_height.value = Number(element_height.value) - Number(actual_height.value)
    app_content_height.value = Number(element_height.value) - Number(browser_url_bar_height.value)
    app_content_width.value = window.innerWidth
}

async function write_errors(errors: Array<GkillError>) {
    //TODO エラーメッセージを画面に出力するように
    errors.forEach(error => {
        console.log(error)
    });
}

async function write_messages(messages: Array<GkillMessage>) {
    //TODO メッセージを画面に出力するように
    messages.forEach(message => {
        console.log(message)
    });
}

window.addEventListener('resize', () => {
    resize_content()
})

resize_content()
load_application_config()
</script>
