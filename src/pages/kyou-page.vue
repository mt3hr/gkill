<template>
    <v-app-bar :height="app_title_bar_height" class="app_bar" color="primary" app flat>
        <v-toolbar-title>Kyou</v-toolbar-title>
        <v-spacer />
    </v-app-bar>
    <v-main class="main">
        <KyouView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="hightlight_targets" :is_image_view="is_image_view" :kyou="kyou" :last_added_tag="''"
            :show_checkbox="false" :show_content_only="false" :show_mi_create_time="true"
            :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
            :show_timeis_plaing_end_button="true" @received_errors="write_errors" @received_messages="write_messages" />
        <ApplicationConfigDialog :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height" :app_content_width="app_content_width"
            :is_show="is_show_application_config_dialog" @received_errors="write_errors"
            @received_messages="write_messages" @requested_reload_application_config="load_application_config" />
    </v-main>
</template>
<script lang="ts" setup>
import { GkillAPI } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { type Ref, ref, computed } from 'vue'
import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import KyouView from './views/kyou-view.vue'
import { InfoIdentifier } from '@/classes/datas/info-identifier'
import { Kyou } from '@/classes/datas/kyou'
import { Tag } from '@/classes/datas/tag'
import { Text } from '@/classes/datas/text'
import { TimeIs } from '@/classes/datas/time-is'
import { Kmemo } from '@/classes/datas/kmemo'
import { URLog } from '@/classes/datas/ur-log'
import { Mi } from '@/classes/datas/mi'
import { Lantana } from '@/classes/datas/lantana'
import { Nlog } from '@/classes/datas/nlog'
import { IDFKyou } from '@/classes/datas/idf-kyou'
import { ReKyou } from '@/classes/datas/re-kyou'
import { GitCommitLog } from '@/classes/datas/git-commit-log'
import KyouListView from './views/kyou-list-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

const actual_height: Ref<Number> = ref(0)
const element_height: Ref<Number> = ref(0)
const browser_url_bar_height: Ref<Number> = ref(0)
const app_title_bar_height: Ref<Number> = ref(50)
const app_title_bar_height_px = computed(() => app_title_bar_height.value.toString().concat("px"))
const gkill_api: Ref<GkillAPI> = ref(GkillAPI.get_instance())
const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
const app_content_height: Ref<Number> = ref(0)
const app_content_width: Ref<Number> = ref(0)

const is_show_application_config_dialog: Ref<boolean> = ref(false)
const hightlight_targets: Ref<Array<InfoIdentifier>> = ref(new Array<InfoIdentifier>())
const is_image_view: Ref<boolean> = ref(false)
const kyou: Ref<Kyou> = ref(new Kyou())

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
    })
}

async function write_messages(messages: Array<GkillMessage>) {
    //TODO メッセージを画面に出力するように
    messages.forEach(message => {
        console.log(message)
    })
}

window.addEventListener('resize', () => {
    resize_content()
})

resize_content()
load_application_config()


</script>
