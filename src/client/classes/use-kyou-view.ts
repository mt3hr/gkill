import { computed, watch, type Ref, ref, nextTick, onUnmounted } from 'vue'
import { format_time } from '@/classes/format-date-time'
import { useDelayedLoading } from '@/classes/use-delayed-loading'
import { Kyou } from '@/classes/datas/kyou'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { KyouViewProps } from '@/pages/views/kyou-view-props'
import type { GkillError } from '@/classes/api/gkill-error'
import type KmemoView from '@/pages/views/kmemo-view.vue'
import type KCView from '@/pages/views/kc-view.vue'
import type MiKyouView from '@/pages/views/mi-kyou-view.vue'
import type NlogView from '@/pages/views/nlog-view.vue'
import type LantanaView from '@/pages/views/lantana-view.vue'
import type TimeIsView from '@/pages/views/time-is-view.vue'
import type UrLogView from '@/pages/views/ur-log-view.vue'
import type IdfKyouView from '@/pages/views/idf-kyou-view.vue'
import type ReKyouView from '@/pages/views/re-kyou-view.vue'
import type MiReKyouView from '@/pages/views/mi-re-kyou-view.vue'
import type GitCommitLogView from '@/pages/views/git-commit-log-view.vue'
import type PluginHtmlView from '@/pages/views/plugin-html-view.vue'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'
import { is_kyou_reloading } from '@/classes/kyou-reload'

export function useKyouView(options: {
    props: KyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kmemo_view = ref<InstanceType<typeof KmemoView> | null>(null)
    const kc_view = ref<InstanceType<typeof KCView> | null>(null)
    const mi_view = ref<InstanceType<typeof MiKyouView> | null>(null)
    const nlog_view = ref<InstanceType<typeof NlogView> | null>(null)
    const lantana_view = ref<InstanceType<typeof LantanaView> | null>(null)
    const timeis_view = ref<InstanceType<typeof TimeIsView> | null>(null)
    const urlog_view = ref<InstanceType<typeof UrLogView> | null>(null)
    const idf_kyou_view = ref<InstanceType<typeof IdfKyouView> | null>(null)
    const rekyou_view = ref<InstanceType<typeof ReKyouView> | null>(null)
    const mirekyou_view = ref<InstanceType<typeof MiReKyouView> | null>(null)
    const git_commit_log_view = ref<InstanceType<typeof GitCommitLogView> | null>(null)
    const plugin_html_view = ref<InstanceType<typeof PluginHtmlView> | null>(null)

    // ── State refs ──
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    // 種別データ(typed_*)をこのKyouViewが読んでいる最中か。
    // Kyou側のis_typed_data_loadedを直接見てはいけない。TimeIsViewが再生中TimeIsの
    // 最新化でreload_with_typed_datas()を呼び、親から渡したcloned_kyouのフラグを
    // 途中で倒すことがある。それを読み込み中とみなすとTimeIsView自身がunmount→remountされ、
    // onMountedの再取得が無限に走る。自分が始めた読み込みだけを追う
    const is_typed_datas_loading: Ref<boolean> = ref(!props.kyou.is_typed_data_loaded)

    // ── Lifecycle ──
    onUnmounted(() => {
        cloned_kyou.value.abort_controller.abort()
        cloned_kyou.value.abort_controller = new AbortController()
    })

    // ── Computed ──
    // 中身が入る前のKyou（ReKyou/MiReKyouの参照先を取りに行っている間など）は
    // idが空のまま。日時だけは new Date(0) が入っているので、そのまま出すと
    // 1970/01/01 が一瞬見えてしまう。取得できるまでは日時自体を出さない
    const is_kyou_loaded = computed(() => props.kyou.id !== "")
    const related_time = computed(() => is_kyou_loaded.value ? format_time(props.kyou.related_time) : "")
    const update_time = computed(() => is_kyou_loaded.value ? format_time(props.kyou.update_time) : "")
    const rep_name = computed(() => props.kyou.rep_name)

    // 参照先を取りに行っている最中(idが空)か、種別データがまだ入っていない間は読み込み中。
    // 種別データはどの型でも必ずAPIを1回叩くので、どのKyouにも待ち時間がある
    const is_kyou_loading = computed(() => !is_kyou_loaded.value || is_typed_datas_loading.value)
    // 速く終わる読み込みでスピナーが明滅しないよう、一定時間かかったときだけ出す
    const show_loading_indicator = useDelayedLoading(is_kyou_loading)

    // 保存後の引き直し中。上のis_kyou_loadingと違い、こちらは中身が既にあるので
    // 差し替えずに重ねて出す。引き直しはページ側(reload_kyou)が回していて、
    // 完了時にこのKyouViewのpropsごと差し替わるため、状態はidで購読する
    const is_reloading = computed(() => is_kyou_reloading(props.kyou.id))
    const show_reloading_indicator = useDelayedLoading(is_reloading)

    const kyou_class = computed(() => {
        let highlighted = false
        for (let i = 0; i < props.highlight_targets.length; i++) {
            if (props.highlight_targets[i].id === props.kyou.id
                && props.highlight_targets[i].create_time.getTime() === props.kyou.create_time.getTime()
                && props.highlight_targets[i].update_time.getTime() === props.kyou.update_time.getTime()) {
                highlighted = true
                break
            }
        }
        if (highlighted) {
            return "highlighted_kyou"
        }
        return ""
    })

    // ── Watchers ──
    watch(() => props.kyou, async () => {
        cloned_kyou.value.abort_controller.abort()
        cloned_kyou.value = props.kyou.clone()
        cloned_kyou.value.abort_controller = new AbortController()
        // reloadを待つ間にも中身は無い。ここで同期的に立てておかないと、
        // force_show_latest_kyou_info(ReKyou/MiReKyouの経路)のときだけ
        // スピナーも中身も出ない空白の時間ができる
        is_typed_datas_loading.value = !cloned_kyou.value.is_typed_data_loaded
        if (props.force_show_latest_kyou_info) {
            await cloned_kyou.value.reload(props.force_show_latest_kyou_info);//最新を読み込むためにReload
        }
        load_attached_infos() // awaitしない(watcherをブロックせずバックグラウンドで読み込む)
    })

    // ── Initialization ──
    load_attached_infos() // awaitしない(セットアップをブロックせずバックグラウンドで読み込む)

    // ── Internal helpers ──
    /** 読み込み中フラグを立てたうえで種別データを読む */
    async function load_typed_datas_with_loading(target_kyou: Kyou): Promise<Array<GkillError>> {
        is_typed_datas_loading.value = !target_kyou.is_typed_data_loaded
        try {
            return await target_kyou.load_typed_datas()
        } finally {
            // 読み込み中に別のKyouへ差し替わっていたら、後始末は後発の読み込みに任せる
            if (cloned_kyou.value === target_kyou) {
                is_typed_datas_loading.value = false
            }
        }
    }

    async function load_attached_infos(): Promise<void> {
        // 読み込み中にcloned_kyouが差し替わっても、始めたときの対象へ書き込むようにする
        const target_kyou = cloned_kyou.value
        try {
            const await_promises = new Array<Promise<Array<GkillError>>>()
            try {
                await_promises.push(load_typed_datas_with_loading(target_kyou))
                if (props.show_attached_tags) {
                    await_promises.push(target_kyou.load_attached_tags())
                }
                if (props.show_attached_texts) {
                    await_promises.push(target_kyou.load_attached_texts())
                }
                if (props.show_attached_notifications) {
                    await_promises.push(target_kyou.load_attached_notifications())
                }
                if (props.show_attached_timeis) {
                    await_promises.push(target_kyou.load_attached_timeis())
                }
                await Promise.all(await_promises)
            } catch (err: unknown) {
                // abortは握りつぶす
                if (!(err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request")))) {
                    // abort以外はエラー出力する
                    console.error(err)
                }
            }
        } catch (err: unknown) {
            // abortは握りつぶす
            if (!(err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request")))) {
                // abort以外はエラー出力する
                console.error(err)
            }
        }
    }

    // ── Business logic ──
    async function show_context_menu(e: PointerEvent): Promise<void> {
        if (!props.enable_context_menu) {
            return
        }
        kmemo_view.value?.show_context_menu(e)
        kc_view.value?.show_context_menu(e)
        mi_view.value?.show_context_menu(e)
        nlog_view.value?.show_context_menu(e)
        lantana_view.value?.show_context_menu(e)
        timeis_view.value?.show_context_menu(e)
        urlog_view.value?.show_context_menu(e)
        idf_kyou_view.value?.show_context_menu(e)
        rekyou_view.value?.show_context_menu(e)
        mirekyou_view.value?.show_context_menu(e)
        git_commit_log_view.value?.show_context_menu(e)
        plugin_html_view.value?.show_context_menu(e)
    }

    function show_kyou_dialog(): void {
        if (props.enable_dialog) {
            emits('requested_open_rykv_dialog', 'kyou', cloned_kyou.value)
        }
    }

    // ── Event relay objects ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Template event handlers ──
    function onRootClick(): void {
        nextTick(() => {
            emits('focused_kyou', cloned_kyou.value)
            emits('clicked_kyou', cloned_kyou.value)
        })
    }

    // ── Return ──
    return {
        // Template refs
        kmemo_view,
        kc_view,
        mi_view,
        nlog_view,
        lantana_view,
        timeis_view,
        urlog_view,
        idf_kyou_view,
        rekyou_view,
        mirekyou_view,
        git_commit_log_view,
        plugin_html_view,

        // State
        cloned_kyou,

        // Computed
        related_time,
        update_time,
        rep_name,
        kyou_class,
        is_kyou_loading,
        show_loading_indicator,
        is_reloading,
        show_reloading_indicator,

        // Business logic
        show_context_menu,
        show_kyou_dialog,
        onRootClick,

        // Event relay objects
        crudRelayHandlers,
    }
}

