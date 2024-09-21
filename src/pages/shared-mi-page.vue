<template>
    <v-app-bar class="app_bar" app color="indigo" flat dark :height="app_title_bar_height_px">
        <v-toolbar-title></v-toolbar-title>
        <v-spacer />
    </v-app-bar>
    <v-main class="main">
        <miSharedTaskView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :share_id="share_mi_id" :application_config="new ApplicationConfig()" :gkill_api="gkill_api"
            @received_errors="write_errors" @received_messages="write_messages" />
    </v-main>
</template>

<script lang="ts" setup>
'use strict';
import { computed, ref, type Ref } from 'vue';
import { GkillAPI } from '@/classes/api/gkill-api';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

import miSharedTaskView from './views/mi-shared-task-view.vue';
import { useRoute } from 'vue-router';
import { ApplicationConfig } from '@/classes/datas/config/application-config';

const actual_height: Ref<Number> = ref(0);
const element_height: Ref<Number> = ref(0);
const browser_url_bar_height: Ref<Number> = ref(0);
const app_title_bar_height: Ref<Number> = ref(50);
const app_title_bar_height_px = computed(() => app_title_bar_height.value.toString().concat("px"))
const gkill_api: Ref<GkillAPI> = ref(new GkillAPI());
const app_content_height: Ref<Number> = ref(0);
const app_content_width: Ref<Number> = ref(0);
const share_mi_id = computed(() => useRoute().query.share_id ? useRoute().query.share_id?.toString()!! : "")

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

</script>
