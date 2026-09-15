import { computed, nextTick, onMounted, ref, type Ref } from 'vue'
import { i18n } from '@/i18n'
import { GkillAPI, GkillAPIForSharedKyou } from '@/classes/api/gkill-api'
import { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { GetSharedKyousRequest } from '@/classes/api/req_res/get-shared-kyous-request'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { useRoute } from 'vue-router'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import { useGkillMessageFeed } from '@/classes/use-gkill-message-feed'

export function useSharedPage() {
    const route = useRoute()

    // ── State refs ──
    const share_id = computed(() => route.query.share_id?.toString() ?? '')
    const view_type: Ref<string | null> = ref(null)
    const share_title: Ref<string | null> = ref(null)
    const gkill_api_plane: Ref<GkillAPI | null> = ref(null)
    const gkill_api_for_share: Ref<GkillAPI | null> = ref(null)
    const application_config: Ref<ApplicationConfig | null> = ref(null)
    const is_loading: Ref<boolean> = ref(true)

    const { push_errors, push_messages } = useGkillMessageFeed()

    function write_errors(errors_: Array<GkillError>): void {
        push_errors(errors_)
    }

    function write_messages(messages_: Array<GkillMessage>): void {
        push_messages(messages_)
    }

    // ── Helpers ──

    gkill_api_plane.value = GkillAPI.get_instance()

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
            gkill_api_for_shared_kyou.timeiss = res.timeiss
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
            gkill_api_for_shared_kyou.attached_timeis_kyous = res.attached_timeis_kyous
            GkillAPI.set_gkill_api(gkill_api_for_shared_kyou)
            // GkillAPIForSharedKyouを設定ここまで

            gkill_api_for_share.value = gkill_api_for_shared_kyou
            gkill_api_for_shared_kyou.set_shared_id_to_cookie(share_id.value)
            application_config.value = (await gkill_api_for_share.value.get_application_config(new GetApplicationConfigRequest())).application_config
            share_title.value = res.title
            view_type.value = res.view_type
        } catch (e) {
            console.error(e)
            const error = new GkillError()
            error.error_code = GkillErrorCodes.failed_shared_kyous
            error.error_message = i18n.global.t("FAILED_LOAD_MESSAGE")
            write_errors([error])
        } finally {
            // 成功・失敗どちらでも読み込み中を解除する。
            // 以前は成功パスの末尾でしか false にしていなかったため、
            // 無効・失効した共有IDを開くとオーバーレイが出たままになり、
            // 出したエラーもオーバーレイの裏に隠れて利用者に伝わらなかった。
            is_loading.value = false
        }
    }

    // ── Lifecycle ──
    onMounted(async () => {
        await reset_dialog_history()
    })

    // ── Init ──
    nextTick(async () => await load_gkill_api_and_application_config())

    return {
        // State
        share_id,
        view_type,
        share_title,
        gkill_api_for_share,
        application_config,
        is_loading,

        // Event handlers
    }
}
