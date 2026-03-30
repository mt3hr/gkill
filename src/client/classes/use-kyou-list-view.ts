import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import { Kyou } from '@/classes/datas/kyou'
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import type { VVirtualScroll } from 'vuetify/components'
import type { KyouListViewProps } from '@/pages/views/kyou-list-view-props'
import type { KyouListViewEmits } from '@/pages/views/kyou-list-view-emits'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'

export function useKyouListView(options: {
    props: KyouListViewProps,
    emits: KyouListViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_list_view = ref<InstanceType<typeof VVirtualScroll> | null>(null)
    const kyou_list_image_view = ref<InstanceType<typeof VVirtualScroll> | null>(null)

    // ── State refs ──
    const match_kyous_for_image: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
    const is_loading: Ref<boolean> = ref(false)

    // ── Computed ──
    const kyou_height_px = computed(() => props.kyou_height ? props.kyou_height.toString().concat("px") : "0px")
    const footer_height = computed(() => props.show_footer ? 48 : 0)
    const footer_class = computed(() => props.is_focused_list ? 'focused_list' : '')

    // ── Watchers ──
    watch(() => props.query, () => reload())
    watch(() => props.matched_kyous, () => reload())

    // ── CRUD relay handlers ──
    const crudRelayHandlers = {
        'deleted_kyou': (kyou: Kyou) => emits('deleted_kyou', kyou),
        'deleted_tag': (tag: Tag) => emits('deleted_tag', tag),
        'deleted_text': (text: Text) => emits('deleted_text', text),
        'deleted_notification': (notification: Notification) => emits('deleted_notification', notification),
        'registered_kyou': (kyou: Kyou) => emits('registered_kyou', kyou),
        'registered_tag': (tag: Tag) => emits('registered_tag', tag),
        'registered_text': (text: Text) => emits('registered_text', text),
        'registered_notification': (notification: Notification) => emits('registered_notification', notification),
        'updated_kyou': (kyou: Kyou) => emits('updated_kyou', kyou),
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'updated_text': (text: Text) => emits('updated_text', text),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_reload_kyou': (kyou: Kyou) => emits('requested_reload_kyou', kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (kyous: Array<Kyou>, checked: boolean) => emits('requested_update_check_kyous', kyous, checked),
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    // ── Internal helpers ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

    async function reload(): Promise<void> {
        if (props.query.is_image_only) {
            match_kyous_for_image.value.splice(0)
            update_match_kyous_for_image()
        } else {
            match_kyous_for_image.value.splice(0)
        }
    }

    async function update_match_kyous_for_image(): Promise<void> {
        match_kyous_for_image.value.splice(0)
        const match_kyous_for_image_result = new Array<Array<Kyou>>()
        for (let i = 0; props.matched_kyous && i < props.matched_kyous.length;) {
            const kyou_row_list = new Array<Kyou>()
            for (let j = 0; props.matched_kyous && j < props.application_config.rykv_image_list_column_number.valueOf(); j++) {
                if (i < props.matched_kyous.length) {
                    const kyou = props.matched_kyous[i]
                    kyou_row_list.push(kyou)
                    i++
                }
            }
            match_kyous_for_image_result.push(kyou_row_list)
        }
        for (let i = 0; i < match_kyous_for_image_result.length; i++) {
            match_kyous_for_image.value.push(match_kyous_for_image_result[i])
        }
    }

    // ── Exposed methods ──
    async function scroll_to(scroll_top: number): Promise<void> {
        return nextTick(async () => {
            const target_element_id = props.query.query_id.concat(props.query.is_image_only ? "_kyou_image_list_view" : "_kyou_list_view")
            const kyou_list_view_element = document.getElementById(target_element_id)
            const scroll_height = kyou_list_view_element?.querySelector(".v-virtual-scroll__container")?.scrollHeight
            if (!kyou_list_view_element || !scroll_height || scroll_height < scroll_top) {
                nextTick(async () => { // nextTickじゃ動かんかったのでsleepで対応
                    await sleep(50)
                    scroll_to(scroll_top)
                })
                return
            }
            kyou_list_view_element.scrollTop = (scroll_top)
        })
    }

    async function scroll_to_kyou(kyou: Kyou): Promise<boolean> {
        let index = -1;
        for (let i = 0; i < props.matched_kyous.length; i++) {
            const kyou_in_list = props.matched_kyous[i]
            if (kyou_in_list.id === kyou.id) {
                index = i
                break
            }
        }

        if (index === -1) {
            return false
        }
        kyou_list_view.value?.scrollToIndex(index)
        kyou_list_image_view.value?.scrollToIndex(index / props.application_config.rykv_image_list_column_number.valueOf())
        return true
    }

    async function scroll_to_time(time: Date): Promise<boolean> {
        let index = -1;
        for (let i = 0; i < props.matched_kyous.length; i++) {
            const kyou = props.matched_kyous[i]
            if (kyou.related_time.getTime() <= time.getTime()) {
                index = i
                break
            }
        }

        if (index === -1) {
            return false
        }
        kyou_list_view.value?.scrollToIndex(index)
        kyou_list_image_view.value?.scrollToIndex(index / props.application_config.rykv_image_list_column_number.valueOf())
        return true
    }

    function set_loading(loading: boolean): void {
        is_loading.value = loading
    }

    function get_is_loading(): boolean {
        return is_loading.value
    }

    function get_query_id(): string {
        return props.query.query_id
    }

    // ── Template event handlers ──
    function onScrollEnd(e: Event): void {
        e.preventDefault()
        emits('scroll_list', (e.target as HTMLElement)?.scrollTop ?? 0)
    }

    function onClickedListView(): void {
        emits('clicked_list_view')
    }

    function onFocusedKyou(kyou: Kyou): void {
        emits('focused_kyou', kyou)
    }

    function onClickedKyou(kyou: Kyou): void {
        emits('focused_kyou', kyou)
        emits('clicked_kyou', kyou)
    }

    function onRequestedSearch(): void {
        emits('requested_search')
    }

    function onRequestedChangeImageOnly(): void {
        emits('requested_change_is_image_only_view', !props.query.is_image_only)
    }

    function onRequestedChangeFocusKyou(): void {
        emits('requested_change_focus_kyou', !props.query.is_focus_kyou_in_list_view)
    }

    function onRequestedCloseColumn(): void {
        if (props.closable) {
            emits('requested_close_column')
        }
    }

    // ── Return ──
    return {
        // Template refs
        kyou_list_view,
        kyou_list_image_view,

        // State
        match_kyous_for_image,
        is_loading,

        // Computed
        kyou_height_px,
        footer_height,
        footer_class,

        // Exposed methods
        scroll_to,
        scroll_to_kyou,
        scroll_to_time,
        set_loading,
        get_is_loading,
        get_query_id,

        // Template event handlers
        onScrollEnd,
        onClickedListView,
        onFocusedKyou,
        onClickedKyou,
        onRequestedSearch,
        onRequestedChangeImageOnly,
        onRequestedChangeFocusKyou,
        onRequestedCloseColumn,

        // Event relay objects
        crudRelayHandlers,
    }
}
