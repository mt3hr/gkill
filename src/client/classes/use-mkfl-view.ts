import { ref } from 'vue'
import type PlayingTimeIsView from '@/pages/views/playing-time-is-view.vue'
import type { MKFLViewEmits } from '@/pages/views/mkfl-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useMkflView(options: {
    emits: MKFLViewEmits,
    playing_timeis_view: ReturnType<typeof ref<InstanceType<typeof PlayingTimeIsView> | null>>,
}) {
    const { emits, playing_timeis_view } = options

    // ── Methods ──
    async function reload_playing_timeis_view(): Promise<void> {
        playing_timeis_view.value?.reload_list(false)
    }

    // ── Event relay objects ──
    // 中継は必ず build_kyou_view_relay で作る。手書きで並べていたころは
    // requested_reload_kyou / requested_update_check_kyous / requested_open_rykv_dialog の
    // 3件が両方の子で抜けていた（タグ・テキスト・通知の変更が親へ届かない）
    const kftlRelayHandlers = build_kyou_view_relay(emits, {
        // KFTLからの削除は実行中の一覧にも効くので引き直す
        deleted_kyou: (deleted_kyou: Kyou) => {
            reload_playing_timeis_view()
            emits('deleted_kyou', deleted_kyou)
        },
    })

    const playingRelayHandlers = build_kyou_view_relay(emits, {
        deleted_kyou: (deleted_kyou: Kyou) => {
            reload_playing_timeis_view()
            emits('deleted_kyou', deleted_kyou)
        },
        registered_kyou: (registered_kyou: Kyou) => {
            reload_playing_timeis_view()
            emits('registered_kyou', registered_kyou)
        },
        updated_kyou: (updated_kyou: Kyou) => {
            reload_playing_timeis_view()
            emits('updated_kyou', updated_kyou)
        },
    })

    // ── Return ──
    return {
        // Methods
        reload_playing_timeis_view,

        // Event relay objects
        kftlRelayHandlers,
        playingRelayHandlers,
    }
}
