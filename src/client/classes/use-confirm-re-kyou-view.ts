import { type Ref, ref } from 'vue'
import { i18n } from '@/i18n'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { ReKyou } from '@/classes/datas/re-kyou'
import { AddReKyouRequest } from '@/classes/api/req_res/add-re-kyou-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ConfirmReKyouViewProps } from '@/pages/views/confirm-re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'
import type { ComponentRef } from '@/classes/component-ref'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { add_tags_to_target } from '@/classes/kyou-tags'

export function useConfirmReKyouView(options: {
    props: ConfirmReKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_tags_view = ref<ComponentRef | null>(null)

    // ── Confirm unknown tag ──
    const confirm_unknown_tag = useConfirmUnknownTag({ application_config: () => props.application_config })

    // ── State refs ──
    const is_requested_submit = ref(false)
    const show_kyou: Ref<boolean> = ref(true)

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Rekyou logic ──
    //
    // idは呼ぶたびに新しく振るので、ガードが無いと連打したぶんだけリポストができる。
    // 何があってもダイアログを閉じるのは use-confirm-delete-kyou-view.ts と同じ理由
    async function rekyou(): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        // タグツリーに無いタグ名なら、リポストする前に確認を取る。
        // ここでは is_requested_submit を立てない ―― 確認ダイアログを開いたまま
        // ボタンが押せなくなるのを避けるため、実行の直前で立てる
        const tag_names = kyou_tags_view.value?.get_tag_names() ?? []
        const unknown_tags = confirm_unknown_tag.collect_unknown_tags(tag_names)
        if (unknown_tags.length !== 0) {
            confirm_unknown_tag.open_confirm(unknown_tags)
            return
        }
        await execute_rekyou(tag_names)
    }

    function cancel_rekyou(): void {
        confirm_unknown_tag.close_confirm()
    }

    async function confirm_rekyou(): Promise<void> {
        confirm_unknown_tag.close_confirm()
        // 確認ダイアログは非モーダルなので、確認中にタグ欄を書き換えられる。取り直す
        await execute_rekyou(kyou_tags_view.value?.get_tag_names() ?? [])
    }

    async function execute_rekyou(tag_names: Array<string>): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        is_requested_submit.value = true
        try {
            // rekyou情報を用意する
            const new_rekyou = new ReKyou()
            new_rekyou.id = props.gkill_api.generate_uuid()
            new_rekyou.is_deleted = false
            new_rekyou.target_id = props.kyou.id
            new_rekyou.related_time = new Date(Date.now())
            new_rekyou.create_app = "gkill"
            new_rekyou.create_device = props.application_config.device
            new_rekyou.create_time = new Date(Date.now())
            new_rekyou.create_user = props.application_config.user_id
            new_rekyou.update_app = "gkill"
            new_rekyou.update_device = props.application_config.device
            new_rekyou.update_time = new Date(Date.now())
            new_rekyou.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_rekyou.id)
            const req = new AddReKyouRequest()
            req.want_response_kyou = true
            req.rekyou = new_rekyou
            const res = await props.gkill_api.add_rekyou(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }

            // タグは registered_kyou より必ず先に付ける。
            // 先に emit すると、タグで絞り込んだ列が空のタグ列を見て「一致しない」と判定し、
            // エラーも出ないまま行が現れない
            const tag_result = await add_tags_to_target(props.gkill_api, props.application_config, new_rekyou.id, tag_names)
            tag_result.added_tags.forEach(added_tag => emits('registered_tag', added_tag))
            if (tag_result.messages.length !== 0) {
                emits('received_messages', tag_result.messages)
            }
            if (tag_result.errors.length !== 0) {
                emits('received_errors', tag_result.errors)
            }

            // 他のadd系と同じく、作ったものを一覧へ反映させる。
            // 列へは局所挿入されるので、Kyouが返らなかったときだけ引き直しへ落とす
            if (res.added_kyou) {
                emits('registered_kyou', res.added_kyou)
            } else {
                emits('requested_reload_list')
            }
        } catch (err: unknown) {
            console.error(err)
            const error = new GkillError()
            error.error_code = GkillErrorCodes.failed_add_rekyou
            error.error_message = i18n.global.t("FAILED_ADD_REKYOU_MESSAGE")
            emits('received_errors', [error])
        } finally {
            is_requested_submit.value = false
            emits('requested_close_dialog')
        }
    }

    // ── Return ──
    return {
        // Template refs
        kyou_tags_view,
        confirm_unknown_tag_dialog: confirm_unknown_tag.confirm_unknown_tag_dialog,

        // Confirm unknown tag
        unknown_tags: confirm_unknown_tag.unknown_tags,
        cancel_rekyou,
        confirm_rekyou,

        // State
        is_requested_submit,
        show_kyou,

        // Methods
        rekyou,

        // Event relay objects
        crudRelayHandlers,
    }
}

