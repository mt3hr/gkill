import { ref, watch } from 'vue'
import type { KyouHistoriesViewProps } from '@/pages/views/kyou-histories-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useKyouHistoriesView(options: {
    props: KyouHistoriesViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const cloned_kyou = ref(new Kyou())

    // ── Init ──
    load_cloned_kyou()

    // ── Watchers ──
    watch(() => props.kyou, () => load_cloned_kyou())

    // ── Business logic ──
    async function load_cloned_kyou() {
        const cloned_kyou_value = props.kyou.clone()
        await cloned_kyou_value.load_attached_histories()
        // ループ条件は新しく読み直した cloned_kyou_value を見ること。
        // 古い ref (cloned_kyou.value) を見ていたときは初期値が new Kyou()
        // (attached_histories が空) なのでループが1度も回らず、related_time の
        // 付け替えが効いていなかった
        for (let i = 0; i < cloned_kyou_value.attached_histories.length; i++) {
            cloned_kyou_value.attached_histories[i].related_time = cloned_kyou_value.attached_histories[i].update_time
        }
        cloned_kyou.value = cloned_kyou_value
    }

    // ── Event relay objects ──
    // このダイアログの中からKyouを編集できるので、編集したら履歴一覧を引き直す。
    // requested_reload_kyou では引き直さない ―― タグ/テキスト/通知の追加では
    // 履歴が増えないので、毎回引くと無駄な get_kyou が1往復増えるだけになる
    const crudRelayHandlers = build_kyou_view_relay(emits, {
        'updated_kyou': (kyou: Kyou) => {
            if (kyou.id === props.kyou.id) {
                load_cloned_kyou()
            }
            emits('updated_kyou', kyou)
        },
        'deleted_kyou': (kyou: Kyou) => {
            if (kyou.id === props.kyou.id) {
                load_cloned_kyou()
            }
            emits('deleted_kyou', kyou)
        },
    })

    // ── Return ──
    return {
        // State
        cloned_kyou,

        // Event relay objects
        crudRelayHandlers,
    }
}

