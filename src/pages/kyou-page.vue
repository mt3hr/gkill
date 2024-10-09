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
//TODO コメントアウト解除 load_application_config()


//TODO 以下テストデータ
const test_kyou_kmemo = new Kyou()
const test_kyou_urlog = new Kyou()
const test_kyou_mi = new Kyou()
const test_kyou_lantana = new Kyou()
const test_kyou_nlog = new Kyou()
const test_kyou_timeis = new Kyou()
const test_kyou_idf = new Kyou()
const test_attached_tag1 = new Tag()
const test_attached_tag2 = new Tag()
const test_attached_text1 = new Text()
const test_attached_text2 = new Text()
const test_attached_timeis1 = new TimeIs()
const test_attached_timeis2 = new TimeIs()
const test_attached_timeis_kyou1 = new Kyou()
const test_attached_timeis_kyou2 = new Kyou()
const kmemo = new Kmemo()
const urlog = new URLog()
const mi = new Mi()
const lantana = new Lantana()
const nlog = new Nlog()
const timeis = new TimeIs()
const idf = new IDFKyou()
const test_kyou_kmemo_for_history = new Kyou()
const test_kyou_urlog_for_history = new Kyou()
const test_kyou_mi_for_history = new Kyou()
const test_kyou_lantana_for_history = new Kyou()
const test_kyou_nlog_for_history = new Kyou()
const test_kyou_timeis_for_history = new Kyou()
const test_kyou_idf_for_history = new Kyou()
const test_attached_tag1_for_history = new Tag()
const test_attached_tag2_for_history = new Tag()
const test_attached_text1_for_history = new Text()
const test_attached_text2_for_history = new Text()
const test_attached_timeis1_for_history = new TimeIs()
const test_attached_timeis2_for_history = new TimeIs()
const kmemo_for_history = new Kmemo()
const urlog_for_history = new URLog()
const mi_for_history = new Mi()
const lantana_for_history = new Lantana()
const nlog_for_history = new Nlog()
const timeis_for_history = new TimeIs()
const idf_for_history = new IDFKyou()

kmemo.content = "テスト・テスト。\nテストKmemo"
kmemo.data_type = "kmemo"
kmemo.rep_name = "Kmemo"
kmemo.create_app = "gkill"
kmemo.create_device = "X1Yoga"
kmemo.create_time = new Date(Date.now())
kmemo.create_user = "mt3hr"
kmemo.id = "95552055-266a-4d41-b9b1-949c719b61f6"
kmemo.related_time = new Date(Date.now())
kmemo.update_app = "gkill"
kmemo.update_device = "X1Yoga"
kmemo.update_time = new Date(Date.now())
kmemo.update_user = "mt3hr"
test_kyou_kmemo.data_type = "kmemo"
test_kyou_kmemo.rep_name = "Kmemo"
test_kyou_kmemo.image_source = ""
test_kyou_kmemo.create_app = "gkill"
test_kyou_kmemo.create_device = "X1Yoga"
test_kyou_kmemo.create_time = new Date(Date.now())
test_kyou_kmemo.create_user = "mt3hr"
test_kyou_kmemo.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_kmemo.related_time = new Date(Date.now())
test_kyou_kmemo.update_app = "gkill"
test_kyou_kmemo.update_device = "X1Yoga"
test_kyou_kmemo.update_time = new Date(Date.now())
test_kyou_kmemo.update_user = "mt3hr"
test_kyou_kmemo.typed_kmemo = kmemo
kmemo_for_history.content = "テスト・テスト。\nテストkmemo_for_history"
kmemo_for_history.data_type = "kmemo_for_history"
kmemo_for_history.rep_name = "kmemo_for_history"
kmemo_for_history.create_app = "gkill"
kmemo_for_history.create_device = "X1Yoga"
kmemo_for_history.create_time = new Date(Date.now())
kmemo_for_history.create_user = "mt3hr"
kmemo_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
kmemo_for_history.related_time = new Date(Date.now())
kmemo_for_history.update_app = "gkill"
kmemo_for_history.update_device = "X1Yoga"
kmemo_for_history.update_time = new Date(Date.now())
kmemo_for_history.update_user = "mt3hr"
test_kyou_kmemo_for_history.data_type = "kmemo_for_history"
test_kyou_kmemo_for_history.rep_name = "kmemo_for_history"
test_kyou_kmemo_for_history.image_source = ""
test_kyou_kmemo_for_history.create_app = "gkill"
test_kyou_kmemo_for_history.create_device = "X1Yoga"
test_kyou_kmemo_for_history.create_time = new Date(Date.now())
test_kyou_kmemo_for_history.create_user = "mt3hr"
test_kyou_kmemo_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_kmemo_for_history.related_time = new Date(Date.now())
test_kyou_kmemo_for_history.update_app = "gkill"
test_kyou_kmemo_for_history.update_device = "X1Yoga"
test_kyou_kmemo_for_history.update_time = new Date(Date.now())
test_kyou_kmemo_for_history.update_user = "mt3hr"
test_kyou_kmemo_for_history.typed_kmemo = kmemo_for_history
urlog.description = "ディスクリプション"
urlog.favicon_image = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAq0lEQVQ4jaWT0Q2DMAxEXyIGYAMygkdgBTZgRDaAETpC2CAbXD+giKoJbYklK4niO8tn20mixnwVGmiOm3MtYPurL8Qv+/lASgBIQjAK9KePkkBgN8AvNw+ECgnMn+r+tBihL8kBQLjuQtfBPMM0QQjZkN/a2LbFr2uCdYVh2MqIMRvSAI8igRmkdJUieiBPDd/AsA1U3SC5Y5neR9mAnHLLKXMCTgQ3rXobnzl8hRUj722/AAAAAElFTkSuQmCC"
urlog.thumbnail_image = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAq0lEQVQ4jaWT0Q2DMAxEXyIGYAMygkdgBTZgRDaAETpC2CAbXD+giKoJbYklK4niO8tn20mixnwVGmiOm3MtYPurL8Qv+/lASgBIQjAK9KePkkBgN8AvNw+ECgnMn+r+tBihL8kBQLjuQtfBPMM0QQjZkN/a2LbFr2uCdYVh2MqIMRvSAI8igRmkdJUieiBPDd/AsA1U3SC5Y5neR9mAnHLLKXMCTgQ3rXobnzl8hRUj722/AAAAAElFTkSuQmCC"
urlog.title = "urlogテスト"
urlog.url = "https://www.youtube.com/"
urlog.data_type = "urlog"
urlog.rep_name = "urlog"
urlog.create_app = "gkill"
urlog.create_device = "X1Yoga"
urlog.create_time = new Date(Date.now())
urlog.create_user = "mt3hr"
urlog.id = "95552055-266a-4d41-b9b1-949c719b61f6"
urlog.related_time = new Date(Date.now())
urlog.update_app = "gkill"
urlog.update_device = "X1Yoga"
urlog.update_time = new Date(Date.now())
urlog.update_user = "mt3hr"
test_kyou_urlog.data_type = "urlog"
test_kyou_urlog.rep_name = "urlog"
test_kyou_urlog.image_source = ""
test_kyou_urlog.create_app = "gkill"
test_kyou_urlog.create_device = "X1Yoga"
test_kyou_urlog.create_time = new Date(Date.now())
test_kyou_urlog.create_user = "mt3hr"
test_kyou_urlog.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_urlog.related_time = new Date(Date.now())
test_kyou_urlog.update_app = "gkill"
test_kyou_urlog.update_device = "X1Yoga"
test_kyou_urlog.update_time = new Date(Date.now())
test_kyou_urlog.update_user = "mt3hr"
test_kyou_urlog.typed_urlog = urlog
urlog.description = "ディスクリプション"
urlog.favicon_image = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAq0lEQVQ4jaWT0Q2DMAxEXyIGYAMygkdgBTZgRDaAETpC2CAbXD+giKoJbYklK4niO8tn20mixnwVGmiOm3MtYPurL8Qv+/lASgBIQjAK9KePkkBgN8AvNw+ECgnMn+r+tBihL8kBQLjuQtfBPMM0QQjZkN/a2LbFr2uCdYVh2MqIMRvSAI8igRmkdJUieiBPDd/AsA1U3SC5Y5neR9mAnHLLKXMCTgQ3rXobnzl8hRUj722/AAAAAElFTkSuQmCC"
urlog.thumbnail_image = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAq0lEQVQ4jaWT0Q2DMAxEXyIGYAMygkdgBTZgRDaAETpC2CAbXD+giKoJbYklK4niO8tn20mixnwVGmiOm3MtYPurL8Qv+/lASgBIQjAK9KePkkBgN8AvNw+ECgnMn+r+tBihL8kBQLjuQtfBPMM0QQjZkN/a2LbFr2uCdYVh2MqIMRvSAI8igRmkdJUieiBPDd/AsA1U3SC5Y5neR9mAnHLLKXMCTgQ3rXobnzl8hRUj722/AAAAAElFTkSuQmCC"
urlog.title = "urlogテスト"
urlog.url = "https://www.youtube.com/"
urlog.data_type = "urlog"
urlog.rep_name = "urlog"
urlog.create_app = "gkill"
urlog.create_device = "X1Yoga"
urlog.create_time = new Date(Date.now())
urlog.create_user = "mt3hr"
urlog.id = "95552055-266a-4d41-b9b1-949c719b61f6"
urlog.related_time = new Date(Date.now())
urlog.update_app = "gkill"
urlog.update_device = "X1Yoga"
urlog.update_time = new Date(Date.now())
urlog.update_user = "mt3hr"
test_kyou_urlog.data_type = "urlog"
test_kyou_urlog.rep_name = "urlog"
test_kyou_urlog.image_source = ""
test_kyou_urlog.create_app = "gkill"
test_kyou_urlog.create_device = "X1Yoga"
test_kyou_urlog.create_time = new Date(Date.now())
test_kyou_urlog.create_user = "mt3hr"
test_kyou_urlog.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_urlog.related_time = new Date(Date.now())
test_kyou_urlog.update_app = "gkill"
test_kyou_urlog.update_device = "X1Yoga"
test_kyou_urlog.update_time = new Date(Date.now())
test_kyou_urlog.update_user = "mt3hr"
test_kyou_urlog.typed_urlog = urlog
urlog_for_history.description = "ディスクリプション"
urlog_for_history.favicon_image = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAq0lEQVQ4jaWT0Q2DMAxEXyIGYAMygkdgBTZgRDaAETpC2CAbXD+giKoJbYklK4niO8tn20mixnwVGmiOm3MtYPurL8Qv+/lASgBIQjAK9KePkkBgN8AvNw+ECgnMn+r+tBihL8kBQLjuQtfBPMM0QQjZkN/a2LbFr2uCdYVh2MqIMRvSAI8igRmkdJUieiBPDd/AsA1U3SC5Y5neR9mAnHLLKXMCTgQ3rXobnzl8hRUj722/AAAAAElFTkSuQmCC"
urlog_for_history.thumbnail_image = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAq0lEQVQ4jaWT0Q2DMAxEXyIGYAMygkdgBTZgRDaAETpC2CAbXD+giKoJbYklK4niO8tn20mixnwVGmiOm3MtYPurL8Qv+/lASgBIQjAK9KePkkBgN8AvNw+ECgnMn+r+tBihL8kBQLjuQtfBPMM0QQjZkN/a2LbFr2uCdYVh2MqIMRvSAI8igRmkdJUieiBPDd/AsA1U3SC5Y5neR9mAnHLLKXMCTgQ3rXobnzl8hRUj722/AAAAAElFTkSuQmCC"
urlog_for_history.title = "urlog_for_historyテスト"
urlog_for_history.url = "https://www.youtube.com/"
urlog_for_history.data_type = "urlog_for_history"
urlog_for_history.rep_name = "urlog_for_history"
urlog_for_history.create_app = "gkill"
urlog_for_history.create_device = "X1Yoga"
urlog_for_history.create_time = new Date(Date.now())
urlog_for_history.create_user = "mt3hr"
urlog_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
urlog_for_history.related_time = new Date(Date.now())
urlog_for_history.update_app = "gkill"
urlog_for_history.update_device = "X1Yoga"
urlog_for_history.update_time = new Date(Date.now())
urlog_for_history.update_user = "mt3hr"
test_kyou_urlog_for_history.data_type = "urlog_for_history"
test_kyou_urlog_for_history.rep_name = "urlog_for_history"
test_kyou_urlog_for_history.image_source = ""
test_kyou_urlog_for_history.create_app = "gkill"
test_kyou_urlog_for_history.create_device = "X1Yoga"
test_kyou_urlog_for_history.create_time = new Date(Date.now())
test_kyou_urlog_for_history.create_user = "mt3hr"
test_kyou_urlog_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_urlog_for_history.related_time = new Date(Date.now())
test_kyou_urlog_for_history.update_app = "gkill"
test_kyou_urlog_for_history.update_device = "X1Yoga"
test_kyou_urlog_for_history.update_time = new Date(Date.now())
test_kyou_urlog_for_history.update_user = "mt3hr"
test_kyou_urlog_for_history.typed_urlog = urlog_for_history
mi.estimate_end_time = new Date(Date.now())
mi.estimate_start_time = new Date(Date.now())
mi.limit_time = new Date(Date.now())
mi.is_checked = false
mi.title = "miテストタスクタスク"
mi.data_type = "mi"
mi.rep_name = "mi"
mi.create_app = "gkill"
mi.create_device = "X1Yoga"
mi.create_time = new Date(Date.now())
mi.create_user = "mt3hr"
mi.id = "95552055-266a-4d41-b9b1-949c719b61f6"
mi.related_time = new Date(Date.now())
mi.update_app = "gkill"
mi.update_device = "X1Yoga"
mi.update_time = new Date(Date.now())
mi.update_user = "mt3hr"
mi.board_name = "hoge"
test_kyou_mi.data_type = "mi"
test_kyou_mi.rep_name = "mi"
test_kyou_mi.image_source = ""
test_kyou_mi.create_app = "gkill"
test_kyou_mi.create_device = "X1Yoga"
test_kyou_mi.create_time = new Date(Date.now())
test_kyou_mi.create_user = "mt3hr"
test_kyou_mi.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_mi.related_time = new Date(Date.now())
test_kyou_mi.update_app = "gkill"
test_kyou_mi.update_device = "X1Yoga"
test_kyou_mi.update_time = new Date(Date.now())
test_kyou_mi.update_user = "mt3hr"
test_kyou_mi.typed_mi = mi
mi_for_history.estimate_end_time = new Date(Date.now())
mi_for_history.estimate_start_time = new Date(Date.now())
mi_for_history.limit_time = new Date(Date.now())
mi_for_history.is_checked = false
mi_for_history.title = "mi_for_historyテストタスクタスク"
mi_for_history.data_type = "mi_for_history"
mi_for_history.rep_name = "mi_for_history"
mi_for_history.create_app = "gkill"
mi_for_history.create_device = "X1Yoga"
mi_for_history.create_time = new Date(Date.now())
mi_for_history.create_user = "mt3hr"
mi_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
mi_for_history.related_time = new Date(Date.now())
mi_for_history.update_app = "gkill"
mi_for_history.update_device = "X1Yoga"
mi_for_history.update_time = new Date(Date.now())
mi_for_history.update_user = "mt3hr"
test_kyou_mi_for_history.data_type = "mi_for_history"
test_kyou_mi_for_history.rep_name = "mi_for_history"
test_kyou_mi_for_history.image_source = ""
test_kyou_mi_for_history.create_app = "gkill"
test_kyou_mi_for_history.create_device = "X1Yoga"
test_kyou_mi_for_history.create_time = new Date(Date.now())
test_kyou_mi_for_history.create_user = "mt3hr"
test_kyou_mi_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_mi_for_history.related_time = new Date(Date.now())
test_kyou_mi_for_history.update_app = "gkill"
test_kyou_mi_for_history.update_device = "X1Yoga"
test_kyou_mi_for_history.update_time = new Date(Date.now())
test_kyou_mi_for_history.update_user = "mt3hr"
test_kyou_mi_for_history.typed_mi = mi_for_history
lantana.mood = 3
lantana.data_type = "lantana"
lantana.rep_name = "lantana"
lantana.create_app = "gkill"
lantana.create_device = "X1Yoga"
lantana.create_time = new Date(Date.now())
lantana.create_user = "mt3hr"
lantana.id = "95552055-266a-4d41-b9b1-949c719b61f6"
lantana.related_time = new Date(Date.now())
lantana.update_app = "gkill"
lantana.update_device = "X1Yoga"
lantana.update_time = new Date(Date.now())
lantana.update_user = "mt3hr"
test_kyou_lantana.data_type = "lantana"
test_kyou_lantana.rep_name = "lantana"
test_kyou_lantana.image_source = ""
test_kyou_lantana.create_app = "gkill"
test_kyou_lantana.create_device = "X1Yoga"
test_kyou_lantana.create_time = new Date(Date.now())
test_kyou_lantana.create_user = "mt3hr"
test_kyou_lantana.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_lantana.related_time = new Date(Date.now())
test_kyou_lantana.update_app = "gkill"
test_kyou_lantana.update_device = "X1Yoga"
test_kyou_lantana.update_time = new Date(Date.now())
test_kyou_lantana.update_user = "mt3hr"
test_kyou_lantana.typed_lantana = lantana
lantana_for_history.mood = 3
lantana_for_history.data_type = "lantana_for_history"
lantana_for_history.rep_name = "lantana_for_history"
lantana_for_history.create_app = "gkill"
lantana_for_history.create_device = "X1Yoga"
lantana_for_history.create_time = new Date(Date.now())
lantana_for_history.create_user = "mt3hr"
lantana_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
lantana_for_history.related_time = new Date(Date.now())
lantana_for_history.update_app = "gkill"
lantana_for_history.update_device = "X1Yoga"
lantana_for_history.update_time = new Date(Date.now())
lantana_for_history.update_user = "mt3hr"
test_kyou_lantana_for_history.data_type = "lantana_for_history"
test_kyou_lantana_for_history.rep_name = "lantana_for_history"
test_kyou_lantana_for_history.image_source = ""
test_kyou_lantana_for_history.create_app = "gkill"
test_kyou_lantana_for_history.create_device = "X1Yoga"
test_kyou_lantana_for_history.create_time = new Date(Date.now())
test_kyou_lantana_for_history.create_user = "mt3hr"
test_kyou_lantana_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_lantana_for_history.related_time = new Date(Date.now())
test_kyou_lantana_for_history.update_app = "gkill"
test_kyou_lantana_for_history.update_device = "X1Yoga"
test_kyou_lantana_for_history.update_time = new Date(Date.now())
test_kyou_lantana_for_history.update_user = "mt3hr"
test_kyou_lantana_for_history.typed_lantana = lantana_for_history
nlog.amount = 200
nlog.shop = "新宿駅"
nlog.title = "Suicaチャージ"
nlog.data_type = "nlog"
nlog.rep_name = "nlog"
nlog.create_app = "gkill"
nlog.create_device = "X1Yoga"
nlog.create_time = new Date(Date.now())
nlog.create_user = "mt3hr"
nlog.id = "95552055-266a-4d41-b9b1-949c719b61f6"
nlog.related_time = new Date(Date.now())
nlog.update_app = "gkill"
nlog.update_device = "X1Yoga"
nlog.update_time = new Date(Date.now())
nlog.update_user = "mt3hr"
test_kyou_nlog.data_type = "nlog"
test_kyou_nlog.rep_name = "nlog"
test_kyou_nlog.image_source = ""
test_kyou_nlog.create_app = "gkill"
test_kyou_nlog.create_device = "X1Yoga"
test_kyou_nlog.create_time = new Date(Date.now())
test_kyou_nlog.create_user = "mt3hr"
test_kyou_nlog.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_nlog.related_time = new Date(Date.now())
test_kyou_nlog.update_app = "gkill"
test_kyou_nlog.update_device = "X1Yoga"
test_kyou_nlog.update_time = new Date(Date.now())
test_kyou_nlog.update_user = "mt3hr"
test_kyou_nlog.typed_nlog = nlog
nlog_for_history.amount = 200
nlog_for_history.shop = "新宿駅"
nlog_for_history.title = "Suicaチャージ"
nlog_for_history.data_type = "nlog_for_history"
nlog_for_history.rep_name = "nlog_for_history"
nlog_for_history.create_app = "gkill"
nlog_for_history.create_device = "X1Yoga"
nlog_for_history.create_time = new Date(Date.now())
nlog_for_history.create_user = "mt3hr"
nlog_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
nlog_for_history.related_time = new Date(Date.now())
nlog_for_history.update_app = "gkill"
nlog_for_history.update_device = "X1Yoga"
nlog_for_history.update_time = new Date(Date.now())
nlog_for_history.update_user = "mt3hr"
test_kyou_nlog_for_history.data_type = "nlog_for_history"
test_kyou_nlog_for_history.rep_name = "nlog_for_history"
test_kyou_nlog_for_history.image_source = ""
test_kyou_nlog_for_history.create_app = "gkill"
test_kyou_nlog_for_history.create_device = "X1Yoga"
test_kyou_nlog_for_history.create_time = new Date(Date.now())
test_kyou_nlog_for_history.create_user = "mt3hr"
test_kyou_nlog_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_nlog_for_history.related_time = new Date(Date.now())
test_kyou_nlog_for_history.update_app = "gkill"
test_kyou_nlog_for_history.update_device = "X1Yoga"
test_kyou_nlog_for_history.update_time = new Date(Date.now())
test_kyou_nlog_for_history.update_user = "mt3hr"
test_kyou_nlog_for_history.typed_nlog = nlog_for_history
timeis.start_time = new Date("2024-01-01 00:00:00")
timeis.end_time = new Date(Date.now())
timeis.title = "開発"
timeis.data_type = "timeis"
timeis.rep_name = "timeis"
timeis.create_app = "gkill"
timeis.create_device = "X1Yoga"
timeis.create_time = new Date(Date.now())
timeis.create_user = "mt3hr"
timeis.id = "95552055-266a-4d41-b9b1-949c719b61f6"
timeis.related_time = new Date(Date.now())
timeis.update_app = "gkill"
timeis.update_device = "X1Yoga"
timeis.update_time = new Date(Date.now())
timeis.update_user = "mt3hr"
test_kyou_timeis.data_type = "timeis"
test_kyou_timeis.rep_name = "timeis"
test_kyou_timeis.image_source = ""
test_kyou_timeis.create_app = "gkill"
test_kyou_timeis.create_device = "X1Yoga"
test_kyou_timeis.create_time = new Date(Date.now())
test_kyou_timeis.create_user = "mt3hr"
test_kyou_timeis.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_timeis.related_time = new Date(Date.now())
test_kyou_timeis.update_app = "gkill"
test_kyou_timeis.update_device = "X1Yoga"
test_kyou_timeis.update_time = new Date(Date.now())
test_kyou_timeis.update_user = "mt3hr"
test_kyou_timeis.typed_timeis = timeis
timeis_for_history.start_time = new Date(Date.now())
timeis_for_history.end_time = new Date(Date.now())
timeis_for_history.title = "開発"
timeis_for_history.data_type = "timeis_for_history"
timeis_for_history.rep_name = "timeis_for_history"
timeis_for_history.create_app = "gkill"
timeis_for_history.create_device = "X1Yoga"
timeis_for_history.create_time = new Date(Date.now())
timeis_for_history.create_user = "mt3hr"
timeis_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
timeis_for_history.related_time = new Date(Date.now())
timeis_for_history.update_app = "gkill"
timeis_for_history.update_device = "X1Yoga"
timeis_for_history.update_time = new Date(Date.now())
timeis_for_history.update_user = "mt3hr"
test_kyou_timeis_for_history.data_type = "timeis_for_history"
test_kyou_timeis_for_history.rep_name = "timeis_for_history"
test_kyou_timeis_for_history.image_source = ""
test_kyou_timeis_for_history.create_app = "gkill"
test_kyou_timeis_for_history.create_device = "X1Yoga"
test_kyou_timeis_for_history.create_time = new Date(Date.now())
test_kyou_timeis_for_history.create_user = "mt3hr"
test_kyou_timeis_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_timeis_for_history.related_time = new Date(Date.now())
test_kyou_timeis_for_history.update_app = "gkill"
test_kyou_timeis_for_history.update_device = "X1Yoga"
test_kyou_timeis_for_history.update_time = new Date(Date.now())
test_kyou_timeis_for_history.update_user = "mt3hr"
test_kyou_timeis_for_history.typed_timeis = timeis_for_history
idf.file_name = "交通費.xlsx"
idf.file_url = "https://www.youtube.com/"
idf.data_type = "idf"
idf.rep_name = "idf"
idf.create_app = "gkill"
idf.create_device = "X1Yoga"
idf.create_time = new Date(Date.now())
idf.create_user = "mt3hr"
idf.id = "95552055-266a-4d41-b9b1-949c719b61f6"
idf.related_time = new Date(Date.now())
idf.update_app = "gkill"
idf.update_device = "X1Yoga"
idf.update_time = new Date(Date.now())
idf.update_user = "mt3hr"
test_kyou_idf.data_type = "idf"
test_kyou_idf.rep_name = "idf"
test_kyou_idf.image_source = ""
test_kyou_idf.create_app = "gkill"
test_kyou_idf.create_device = "X1Yoga"
test_kyou_idf.create_time = new Date(Date.now())
test_kyou_idf.create_user = "mt3hr"
test_kyou_idf.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_idf.related_time = new Date(Date.now())
test_kyou_idf.update_app = "gkill"
test_kyou_idf.update_device = "X1Yoga"
test_kyou_idf.update_time = new Date(Date.now())
test_kyou_idf.update_user = "mt3hr"
test_kyou_idf.typed_idf_kyou = idf
idf_for_history.file_name = "交通費.xlsx"
idf_for_history.file_url = "https://www.youtube.com/"
idf_for_history.data_type = "idf_for_history"
idf_for_history.rep_name = "idf_for_history"
idf_for_history.create_app = "gkill"
idf_for_history.create_device = "X1Yoga"
idf_for_history.create_time = new Date(Date.now())
idf_for_history.create_user = "mt3hr"
idf_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
idf_for_history.related_time = new Date(Date.now())
idf_for_history.update_app = "gkill"
idf_for_history.update_device = "X1Yoga"
idf_for_history.update_time = new Date(Date.now())
idf_for_history.update_user = "mt3hr"
test_kyou_idf_for_history.data_type = "idf_for_history"
test_kyou_idf_for_history.rep_name = "idf_for_history"
test_kyou_idf_for_history.image_source = ""
test_kyou_idf_for_history.create_app = "gkill"
test_kyou_idf_for_history.create_device = "X1Yoga"
test_kyou_idf_for_history.create_time = new Date(Date.now())
test_kyou_idf_for_history.create_user = "mt3hr"
test_kyou_idf_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_kyou_idf_for_history.related_time = new Date(Date.now())
test_kyou_idf_for_history.update_app = "gkill"
test_kyou_idf_for_history.update_device = "X1Yoga"
test_kyou_idf_for_history.update_time = new Date(Date.now())
test_kyou_idf_for_history.update_user = "mt3hr"
test_kyou_idf_for_history.typed_idf_kyou = idf_for_history
test_attached_tag1.tag = "タグ1"
test_attached_tag1.create_app = "gkill"
test_attached_tag1.create_device = "X1Yoga"
test_attached_tag1.create_time = new Date(Date.now())
test_attached_tag1.create_user = "mt3hr"
test_attached_tag1.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_tag1.related_time = new Date(Date.now())
test_attached_tag1.update_app = "gkill"
test_attached_tag1.update_device = "X1Yoga"
test_attached_tag1.update_time = new Date(Date.now())
test_attached_tag1.update_user = "mt3hr"
test_attached_tag1_for_history.tag = "タグ1"
test_attached_tag1_for_history.create_app = "gkill"
test_attached_tag1_for_history.create_device = "X1Yoga"
test_attached_tag1_for_history.create_time = new Date(Date.now())
test_attached_tag1_for_history.create_user = "mt3hr"
test_attached_tag1_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_tag1_for_history.related_time = new Date(Date.now())
test_attached_tag1_for_history.update_app = "gkill"
test_attached_tag1_for_history.update_device = "X1Yoga"
test_attached_tag1_for_history.update_time = new Date(Date.now())
test_attached_tag1_for_history.update_user = "mt3hr"
test_attached_tag2.tag = "タグ2"
test_attached_tag2.create_app = "gkill"
test_attached_tag2.create_device = "X1Yoga"
test_attached_tag2.create_time = new Date(Date.now())
test_attached_tag2.create_user = "mt3hr"
test_attached_tag2.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_tag2.related_time = new Date(Date.now())
test_attached_tag2.update_app = "gkill"
test_attached_tag2.update_device = "X1Yoga"
test_attached_tag2.update_time = new Date(Date.now())
test_attached_tag2.update_user = "mt3hr"
test_attached_tag2_for_history.tag = "タグ2"
test_attached_tag2_for_history.create_app = "gkill"
test_attached_tag2_for_history.create_device = "X1Yoga"
test_attached_tag2_for_history.create_time = new Date(Date.now())
test_attached_tag2_for_history.create_user = "mt3hr"
test_attached_tag2_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_tag2_for_history.related_time = new Date(Date.now())
test_attached_tag2_for_history.update_app = "gkill"
test_attached_tag2_for_history.update_device = "X1Yoga"
test_attached_tag2_for_history.update_time = new Date(Date.now())
test_attached_tag2_for_history.update_user = "mt3hr"
test_attached_text1.text = "テキスト\n1\nテスト"
test_attached_text1.create_app = "gkill"
test_attached_text1.create_device = "X1Yoga"
test_attached_text1.create_time = new Date(Date.now())
test_attached_text1.create_user = "mt3hr"
test_attached_text1.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_text1.related_time = new Date(Date.now())
test_attached_text1.update_app = "gkill"
test_attached_text1.update_device = "X1Yoga"
test_attached_text1.update_time = new Date(Date.now())
test_attached_text1.update_user = "mt3hr"
test_attached_text1_for_history.text = "テキスト\n1\nテスト"
test_attached_text1_for_history.create_app = "gkill"
test_attached_text1_for_history.create_device = "X1Yoga"
test_attached_text1_for_history.create_time = new Date(Date.now())
test_attached_text1_for_history.create_user = "mt3hr"
test_attached_text1_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_text1_for_history.related_time = new Date(Date.now())
test_attached_text1_for_history.update_app = "gkill"
test_attached_text1_for_history.update_device = "X1Yoga"
test_attached_text1_for_history.update_time = new Date(Date.now())
test_attached_text1_for_history.update_user = "mt3hr"
test_attached_text2.text = "テキスト\n2\nテスト"
test_attached_text2.create_app = "gkill"
test_attached_text2.create_device = "X1Yoga"
test_attached_text2.create_time = new Date(Date.now())
test_attached_text2.create_user = "mt3hr"
test_attached_text2.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_text2.related_time = new Date(Date.now())
test_attached_text2.update_app = "gkill"
test_attached_text2.update_device = "X1Yoga"
test_attached_text2.update_time = new Date(Date.now())
test_attached_text2.update_user = "mt3hr"
test_attached_text2_for_history.text = "テキスト\n2\nテスト"
test_attached_text2_for_history.create_app = "gkill"
test_attached_text2_for_history.create_device = "X1Yoga"
test_attached_text2_for_history.create_time = new Date(Date.now())
test_attached_text2_for_history.create_user = "mt3hr"
test_attached_text2_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_text2_for_history.related_time = new Date(Date.now())
test_attached_text2_for_history.update_app = "gkill"
test_attached_text2_for_history.update_device = "X1Yoga"
test_attached_text2_for_history.update_time = new Date(Date.now())
test_attached_text2_for_history.update_user = "mt3hr"
test_attached_timeis1.start_time = new Date(Date.now())
test_attached_timeis1.end_time = new Date(Date.now())
test_attached_timeis1.title = "timeisテスト1"
test_attached_timeis1.data_type = "timeis"
test_attached_timeis1.rep_name = "timeis"
test_attached_timeis1.create_app = "gkill"
test_attached_timeis1.create_device = "X1Yoga"
test_attached_timeis1.create_time = new Date(Date.now())
test_attached_timeis1.create_user = "mt3hr"
test_attached_timeis1.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_timeis1.related_time = new Date(Date.now())
test_attached_timeis1.update_app = "gkill"
test_attached_timeis1.update_device = "X1Yoga"
test_attached_timeis1.update_time = new Date(Date.now())
test_attached_timeis1.update_user = "mt3hr"
test_attached_timeis1_for_history.start_time = new Date(Date.now())
test_attached_timeis1_for_history.end_time = new Date(Date.now())
test_attached_timeis1_for_history.title = "timeisテスト1"
test_attached_timeis1_for_history.data_type = "timeis"
test_attached_timeis1_for_history.rep_name = "timeis"
test_attached_timeis1_for_history.create_app = "gkill"
test_attached_timeis1_for_history.create_device = "X1Yoga"
test_attached_timeis1_for_history.create_time = new Date(Date.now())
test_attached_timeis1_for_history.create_user = "mt3hr"
test_attached_timeis1_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_timeis1_for_history.related_time = new Date(Date.now())
test_attached_timeis1_for_history.update_app = "gkill"
test_attached_timeis1_for_history.update_device = "X1Yoga"
test_attached_timeis1_for_history.update_time = new Date(Date.now())
test_attached_timeis1_for_history.update_user = "mt3hr"
test_attached_timeis2.start_time = new Date(Date.now())
test_attached_timeis2.end_time = new Date(Date.now())
test_attached_timeis2.title = "timeisテスト2"
test_attached_timeis2.data_type = "timeis"
test_attached_timeis2.rep_name = "timeis"
test_attached_timeis2.create_app = "gkill"
test_attached_timeis2.create_device = "X1Yoga"
test_attached_timeis2.create_time = new Date(Date.now())
test_attached_timeis2.create_user = "mt3hr"
test_attached_timeis2.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_timeis2.related_time = new Date(Date.now())
test_attached_timeis2.update_app = "gkill"
test_attached_timeis2.update_device = "X1Yoga"
test_attached_timeis2.update_time = new Date(Date.now())
test_attached_timeis2.update_user = "mt3hr"
test_attached_timeis2_for_history.start_time = new Date(Date.now())
test_attached_timeis2_for_history.end_time = new Date(Date.now())
test_attached_timeis2_for_history.title = "timeisテスト2"
test_attached_timeis2_for_history.data_type = "timeis"
test_attached_timeis2_for_history.rep_name = "timeis"
test_attached_timeis2_for_history.create_app = "gkill"
test_attached_timeis2_for_history.create_device = "X1Yoga"
test_attached_timeis2_for_history.create_time = new Date(Date.now())
test_attached_timeis2_for_history.create_user = "mt3hr"
test_attached_timeis2_for_history.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_timeis2_for_history.related_time = new Date(Date.now())
test_attached_timeis2_for_history.update_app = "gkill"
test_attached_timeis2_for_history.update_device = "X1Yoga"
test_attached_timeis2_for_history.update_time = new Date(Date.now())
test_attached_timeis2_for_history.update_user = "mt3hr"
test_attached_timeis_kyou1.data_type = "timeis"
test_attached_timeis_kyou1.rep_name = "timeis"
test_attached_timeis_kyou1.create_app = "gkill"
test_attached_timeis_kyou1.create_device = "X1Yoga"
test_attached_timeis_kyou1.create_time = new Date(Date.now())
test_attached_timeis_kyou1.create_user = "mt3hr"
test_attached_timeis_kyou1.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_timeis_kyou1.related_time = new Date(Date.now())
test_attached_timeis_kyou1.update_app = "gkill"
test_attached_timeis_kyou1.update_device = "X1Yoga"
test_attached_timeis_kyou1.update_time = new Date(Date.now())
test_attached_timeis_kyou1.update_user = "mt3hr"
test_attached_timeis_kyou1.typed_timeis = test_attached_timeis1
test_attached_timeis_kyou2.data_type = "timeis"
test_attached_timeis_kyou2.rep_name = "timeis"
test_attached_timeis_kyou2.create_app = "gkill"
test_attached_timeis_kyou2.create_device = "X1Yoga"
test_attached_timeis_kyou2.create_time = new Date(Date.now())
test_attached_timeis_kyou2.create_user = "mt3hr"
test_attached_timeis_kyou2.id = "95552055-266a-4d41-b9b1-949c719b61f6"
test_attached_timeis_kyou2.related_time = new Date(Date.now())
test_attached_timeis_kyou2.update_app = "gkill"
test_attached_timeis_kyou2.update_device = "X1Yoga"
test_attached_timeis_kyou2.update_time = new Date(Date.now())
test_attached_timeis_kyou2.update_user = "mt3hr"
test_attached_timeis_kyou2.typed_timeis = test_attached_timeis2

test_attached_tag1.attached_histories = [test_attached_tag1, test_attached_tag1_for_history]
test_attached_tag2.attached_histories = [test_attached_tag2, test_attached_tag2_for_history]

test_attached_text1.attached_histories = [test_attached_text1, test_attached_text1_for_history]
test_attached_text2.attached_histories = [test_attached_text2, test_attached_text2_for_history]

test_attached_timeis1.attached_histories = [test_attached_timeis1, test_attached_timeis1_for_history]
test_attached_timeis2.attached_histories = [test_attached_timeis2, test_attached_timeis2_for_history]

test_kyou_kmemo.attached_tags = [test_attached_tag1, test_attached_tag2]
test_kyou_kmemo.attached_texts = [test_attached_text1, test_attached_text2]
test_kyou_kmemo.attached_timeis_kyou = [test_attached_timeis_kyou1, test_attached_timeis_kyou2]
test_kyou_kmemo.attached_histories = [test_kyou_kmemo, test_kyou_kmemo_for_history]

test_kyou_urlog.attached_tags = [test_attached_tag1, test_attached_tag2]
test_kyou_urlog.attached_texts = [test_attached_text1, test_attached_text2]
test_kyou_urlog.attached_timeis_kyou = [test_attached_timeis_kyou1, test_attached_timeis_kyou2]
test_kyou_urlog.attached_histories = [test_kyou_kmemo, test_kyou_kmemo_for_history]

test_kyou_mi.attached_tags = [test_attached_tag1, test_attached_tag2]
test_kyou_mi.attached_texts = [test_attached_text1, test_attached_text2]
test_kyou_mi.attached_timeis_kyou = [test_attached_timeis_kyou1, test_attached_timeis_kyou2]
test_kyou_mi.attached_histories = [test_kyou_kmemo, test_kyou_kmemo_for_history]

test_kyou_lantana.attached_tags = [test_attached_tag1, test_attached_tag2]
test_kyou_lantana.attached_texts = [test_attached_text1, test_attached_text2]
test_kyou_lantana.attached_timeis_kyou = [test_attached_timeis_kyou1, test_attached_timeis_kyou2]
test_kyou_lantana.attached_histories = [test_kyou_lantana, test_kyou_lantana_for_history]

test_kyou_nlog.attached_tags = [test_attached_tag1, test_attached_tag2]
test_kyou_nlog.attached_texts = [test_attached_text1, test_attached_text2]
test_kyou_nlog.attached_timeis_kyou = [test_attached_timeis_kyou1, test_attached_timeis_kyou2]
test_kyou_nlog.attached_histories = [test_kyou_nlog, test_kyou_nlog_for_history]

test_kyou_timeis.attached_tags = [test_attached_tag1, test_attached_tag2]
test_kyou_timeis.attached_texts = [test_attached_text1, test_attached_text2]
test_kyou_timeis.attached_timeis_kyou = [test_attached_timeis_kyou1, test_attached_timeis_kyou2]
test_kyou_timeis.attached_histories = [test_kyou_kmemo, test_kyou_kmemo_for_history]

test_kyou_idf.attached_tags = [test_attached_tag1, test_attached_tag2]
test_kyou_idf.attached_texts = [test_attached_text1, test_attached_text2]
test_kyou_idf.attached_timeis_kyou = [test_attached_timeis_kyou1, test_attached_timeis_kyou2]
test_kyou_idf.attached_histories = [test_kyou_idf, test_kyou_idf_for_history]

kyou.value = test_kyou_urlog
</script>
