'use strict'

import { useFloatingDialog } from "@/classes/use-floating-dialog"
import { ref, type Ref, type ComponentPublicInstance, computed, onBeforeUnmount, watch } from 'vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { KyouListViewDialogProps } from '@/pages/dialogs/kyou-list-view-dialog-props'
import type { KyouListViewEmits } from '@/pages/views/kyou-list-view-emits'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'
import { new_reload_batch, refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'

export function useKyouListViewDialog(options: {
    props: KyouListViewDialogProps,
    emits: KyouListViewEmits,
    model_value: Ref<Array<Kyou> | undefined>,
}) {
    const { props, emits, model_value } = options

    // ── State refs ──
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)

    // このダイアログの中から開いたrykvダイアログ。
    // ページ最上位のRykvDialogHostへ持ち上げてしまうと、そこから出る
    // requested_reload_kyou がページの reload_kyou にしか届かず、
    // このダイアログが抱えているリストには戻ってこない（Vueのイベントは上方向にしか流れない）。
    // だからここで自分でホストする。
    const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])

    // ── Business logic ──
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }

    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    async function reload_kyou(kyou: Kyou): Promise<void> {
        // model_value は親（dnote-item-view の related_kyous /
        // aggregated-list-item の aggregated_item.match_kyous）の配列そのものなので、
        // in-placeで差し替える。新しい配列に差し替えると親と縁が切れる
        // リストと開いているダイアログは同じ更新から派生しているので、
        // 同じ値を渡して1往復に合流させる
        const requested_at = new_reload_batch()
        if (model_value.value) {
            await refresh_kyou_in_list(model_value.value, kyou, { requested_at: requested_at })
        }
        await reload_opened_dialog_kyou(kyou, requested_at)
    }

    async function reload_opened_dialog_kyou(kyou: Kyou, requested_at: number = new_reload_batch()): Promise<void> {
        const target_dialog_ids = opened_dialogs.value
            .filter(opened_dialog => opened_dialog.kyou.id === kyou.id)
            .map(opened_dialog => opened_dialog.id)
        if (target_dialog_ids.length === 0) {
            return
        }
        // 同じKyouに対する引き直しは kyou-reload 側で合流するので、
        // ここで並列に呼んでもリクエストは1本しか飛ばない
        await Promise.all(target_dialog_ids.map(async (dialog_id) => {
            const refreshed = await refresh_kyou(kyou, undefined, requested_at)
            if (!refreshed) {
                return
            }
            replace_opened_dialog_kyou(dialog_id, refreshed)
        }))
    }

    function replace_opened_dialog_kyou(dialog_id: string, kyou: Kyou): void {
        for (let i = 0; i < opened_dialogs.value.length; i++) {
            if (opened_dialogs.value[i].id === dialog_id) {
                opened_dialogs.value[i] = { ...opened_dialogs.value[i], kyou: kyou }
                return
            }
        }
    }

    function open_rykv_dialog(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload): void {
        const dialog_id = props.gkill_api.generate_uuid()
        opened_dialogs.value.push({
            id: dialog_id,
            kind: kind,
            kyou: kyou.clone(),
            payload: payload ?? null,
            opened_at: Date.now(),
        })
        // 開いた直後にも最新化する。リストのKyouは検索時点のものなので、
        // 別経路で更新されていると古い内容で編集ダイアログが開いてしまう
        ;(async (): Promise<void> => {
            const refreshed = await refresh_kyou(kyou)
            if (!refreshed) {
                return
            }
            replace_opened_dialog_kyou(dialog_id, refreshed)
        })().catch((err: unknown) => console.error(err))
    }

    function close_rykv_dialog(dialog_id: string): void {
        for (let i = 0; i < opened_dialogs.value.length; i++) {
            if (opened_dialogs.value[i].id === dialog_id) {
                opened_dialogs.value.splice(i, 1)
                return
            }
        }
    }

    // ── Template event handlers ──
    function onDeletedKyou(deleted_kyou: Kyou): void {
        if (model_value.value) {
            for (let i = model_value.value.length - 1; i >= 0; i--) {
                if (model_value.value[i].id === deleted_kyou.id) {
                    model_value.value.splice(i, 1)
                }
            }
        }
        emits('deleted_kyou', deleted_kyou)
    }

    // ── Event relay objects ──
    //
    // 付随データ(Tag/Text/Notification)のCRUDと新規Kyouは、かつて
    // 「このリストの内容を変えないので」という理由で握りつぶされていたが、これは誤り。
    // 要素数は変えないが、要素の中身（表示される添付タグ等）は変える。
    // 中身の更新を担うのは requested_reload_kyou のほうで、registered_tag 等は
    // 「タグ名一覧が変わった」ことを親（rykvのタグサイドバー等）へ知らせる情報なので、
    // このダイアログ自身は何もしないが親には必ず流す。
    const kyou_relay_overrides = {
        // クリックはフォーカス移動も伴う
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
        'deleted_kyou': (deleted_kyou: Kyou) => onDeletedKyou(deleted_kyou),
        'updated_kyou': (updated_kyou: Kyou) => { reload_kyou(updated_kyou); emits('updated_kyou', updated_kyou) },
        // タグ/テキスト/通知の変更は updated_kyou を出さない。唯一の信号がこれ
        'requested_reload_kyou': (kyou: Kyou) => { reload_kyou(kyou); emits('requested_reload_kyou', kyou) },
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => {
            if (props.host_rykv_dialogs === false) {
                emits('requested_open_rykv_dialog', kind, kyou, payload)
                return
            }
            open_rykv_dialog(kind, kyou, payload)
        },
    }

    const crudRelayHandlers = build_kyou_dialog_relay(emits, kyou_relay_overrides)

    const dialogHostHandlers = {
        ...build_kyou_dialog_relay(emits, kyou_relay_overrides),
        // closed はダイアログホスト固有なので共通束には含まれない
        'closed': (dialog_id: string) => close_rykv_dialog(dialog_id),
    }

    // ── Return ──
    const ui = useFloatingDialog("kyou-list-view-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    // ダイアログはユーザ操作でリサイズされる。useFloatingDialog は外側コンテナに
    // inline width/height を書くだけで子には通知しないので、リストを載せている
    // v-card の実寸を ResizeObserver で測って KyouListView に px で渡す。
    // (KyouListView は v-virtual-scroll(renderless) の表示行数計算に数値の高さが要るため、
    //  CSS の flex 追従だけでは埋まらない)
    const list_card_ref = ref<ComponentPublicInstance | HTMLElement | null>(null)
    const observed_width = ref(0)
    const observed_height = ref(0)
    // kyou-list-view.vue はスクロールコンテナを width + 8 で描くので、その分を差し引く
    const view_width = computed(() => observed_width.value > 0 ? Math.max(200, observed_width.value - 8) : 400)
    const view_height = computed(() => observed_height.value > 0 ? observed_height.value : props.list_height.valueOf())
    function resolve_element(target: ComponentPublicInstance | HTMLElement | null): HTMLElement | null {
            if (!target) return null
            return target instanceof HTMLElement ? target : (target.$el as HTMLElement | null)
    }
    let card_ro: ResizeObserver | null = null
    watch(list_card_ref, (el, old_el) => {
            const old_element = resolve_element(old_el ?? null)
            if (card_ro && old_element) { try { card_ro.unobserve(old_element) } catch { /* noop */ } }
            const element = resolve_element(el)
            if (element) {
                    if (!card_ro) {
                            card_ro = new ResizeObserver((entries) => {
                                    for (const entry of entries) {
                                            observed_width.value = entry.contentRect.width
                                            observed_height.value = entry.contentRect.height
                                    }
                            })
                    }
                    card_ro.observe(element)
            }
    }, { flush: 'post' })
    onBeforeUnmount(() => { card_ro?.disconnect(); card_ro = null })

    return {
        ui,
        list_card_ref,
        view_width,
        view_height,
        // State
        is_show_dialog,
        opened_dialogs,

        // Business logic
        show,
        hide,
        reload_kyou,
        open_rykv_dialog,
        close_rykv_dialog,

        // Template event handlers
        onDeletedKyou,

        // Event relay objects
        crudRelayHandlers,
        dialogHostHandlers,
    }
}
