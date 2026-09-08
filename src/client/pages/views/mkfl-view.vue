<template>
    <div>
        <kftlView :app_content_height="app_content_height.valueOf() / 2" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            v-on="kftlRelayHandlers"
            @saved_kyou_by_kftl="(last_added_request_time: Date) => {
                playing_timeis_view?.set_last_added_request_time(new Date(Math.max(last_added_request_time.getTime(), Date.now())))
                reload_playing_timeis_view()
                emits('saved_kyou_by_kftl', last_added_request_time)
            }" ref="kftl_view" />
        <PlayingTimeIsView :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="(app_content_height.valueOf() / 2) + 4" :app_content_width="app_content_width"
            :is_hosted_in_dialog="is_hosted_in_dialog"
            :kyou_change_channel="null /* 単独ページ。画面間の伝播はポートの中だけ */"
            v-on="playingRelayHandlers"
            ref="playing_timeis_view" />
    </div>
</template>
<script lang="ts" setup>
import { onMounted, ref } from 'vue';
import kftlView from './kftl-view.vue'
import PlayingTimeIsView from './playing-time-is-view.vue'
import type { MKFLProps } from './mkfl-view-props';
import type { MKFLViewEmits } from './mkfl-view-emits';
import { useMkflView } from '@/classes/use-mkfl-view'

const playing_timeis_view = ref<InstanceType<typeof PlayingTimeIsView> | null>(null);
const kftl_view = ref<InstanceType<typeof kftlView> | null>(null);

// 以前はテキストエリアの autofocus 属性で載っていたが、view に autofocus を書くと
// 他画面に埋め込んだときにページ読込でフォーカスを奪うので、置く側から明示的に呼ぶ
onMounted(() => kftl_view.value?.focus_kftl_text_area())

defineProps<MKFLProps>()
const emits = defineEmits<MKFLViewEmits>()

const {
    reload_playing_timeis_view,

    // Event relay objects
    kftlRelayHandlers,
    playingRelayHandlers,
} = useMkflView({ emits, playing_timeis_view })
</script>
