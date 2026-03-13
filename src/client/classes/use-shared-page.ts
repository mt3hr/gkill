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
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'

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

    const messages: Ref<Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>> = ref([])

    // ── Helpers ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

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
            is_loading.value = false
        } catch (e) {
            console.error(e)
            const error = new GkillError()
            error.error_code = GkillErrorCodes.failed_shared_kyous
            error.error_message = i18n.global.t("FAILED_LOAD_MESSAGE")
            write_errors([error])
        }
    }

    function write_errors(errors_: Array<GkillError>): void {
        const received_errors = new Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>()
        for (let i = 0; i < errors_.length; i++) {
            if (errors_[i] && errors_[i].error_message) {
                received_errors.push({
                    code: errors_[i].error_code,
                    message: errors_[i].error_message,
                    id: GkillAPI.get_instance().generate_uuid(),
                    show_snackbar: true,
                    closable: errors_[i].show_keep,
                    auto_close_duration_milli_seconds: errors_[i].show_keep ? null : 2500,
                    is_error: true,
                })
            }
        }
        messages.value.push(...received_errors)
        for (let j = 0; j < received_errors.length; j++) {
            const auto_close_duration_milli_seconds = received_errors[j].auto_close_duration_milli_seconds
            if (auto_close_duration_milli_seconds) {
                sleep(auto_close_duration_milli_seconds).then(() => {
                    close_message(received_errors[j].id)
                })
            }
        }
    }

    function write_messages(messages_: Array<GkillMessage>): void {
        const received_messages = new Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>()
        for (let i = 0; i < messages_.length; i++) {
            if (messages_[i] && messages_[i].message) {
                received_messages.push({
                    code: messages_[i].message_code,
                    message: messages_[i].message,
                    id: GkillAPI.get_instance().generate_uuid(),
                    show_snackbar: true,
                    closable: messages_[i].show_keep,
                    auto_close_duration_milli_seconds: messages_[i].show_keep ? null : 2500,
                    is_error: false,
                })
            }
        }
        messages.value.push(...received_messages)
        for (let j = 0; j < received_messages.length; j++) {
            const auto_close_duration_milli_seconds = received_messages[j].auto_close_duration_milli_seconds
            if (auto_close_duration_milli_seconds) {
                sleep(auto_close_duration_milli_seconds).then(() => {
                    close_message(received_messages[j].id)
                })
            }
        }
    }

    function close_message(message_id: string): void {
        for (let i = 0; i < messages.value.length; i++) {
            if (messages.value[i].id === message_id) {
                messages.value.splice(i, 1)
                return
            }
        }
    }

    // ── Lifecycle ──
    onMounted(async () => {
        await resetDialogHistory()
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
        messages,

        // Event handlers
        close_message,
    }
}
