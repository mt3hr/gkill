import { Kyou } from '@/classes/datas/kyou'
import { computed, nextTick, onUnmounted, type Ref, ref, toRaw, watch } from 'vue'
import type { VVirtualScroll } from 'vuetify/components'
import type { KyouListViewProps } from '@/pages/views/kyou-list-view-props'
import type { KyouListViewEmits } from '@/pages/views/kyou-list-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

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
    const has_loaded: Ref<boolean> = ref(false)

    // ── Computed ──
    const kyou_height_px = computed(() => props.kyou_height ? props.kyou_height.toString().concat("px") : "0px")
    const footer_height = computed(() => props.show_footer ? 48 : 0)
    const footer_class = computed(() => props.is_focused_list ? 'focused_list' : '')

    // ── Business logic ──
    /**
     * 行にrelated_timeを出すか。
     * Mi板で開始・終了・制限のいずれかを含めているとき、作成射影の行では
     * 各Viewが自前で日時を出すのでヘッダのrelated_timeは伏せる。
     * MiReKyouはmirekyou_*、Miはmi_*で来るのでどちらも同じ射影として扱う。
     */
    function should_show_related_time(kyou: Kyou): boolean {
        const is_create_projection = kyou.data_type === 'mi_create' || kyou.data_type === 'mirekyou_create'
        return !(props.query.for_mi && is_create_projection
            && (props.query.include_start_mi || props.query.include_end_mi || props.query.include_limit_mi))
    }

    // ── Watchers ──
    // 3つの監視元は**1つの watch にまとめること**。
    // 別々に張ると、新しい検索が同じtickで3つとも変える(query差し替え・配列差し替え・長さ変化)ので、
    // 画像モードのグリッド組み直しが1回の検索で2〜3回走る。
    // 複数ソースの watch なら Vue が1tickにつき1回へまとめてくれる。
    //
    // 参照監視だけだと、in-placeのsplice(削除・追加時の局所挿入・mi板のD&D)で
    // 画像モードのグリッドが作り直されない。deep監視は30万件になるので使えないため長さも見る
    watch([
        () => props.query,
        () => props.matched_kyous,
        () => props.matched_kyous?.length ?? 0,
    ], () => reload())
    watch(() => props.application_config.rykv_image_list_column_number, () => {
        if (props.query.is_image_only) {
            update_match_kyous_for_image()
        }
    })

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Internal helpers ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

    async function reload(): Promise<void> {
        if (props.query.is_image_only) {
            update_match_kyous_for_image()
        } else {
            match_kyous_for_image.value.splice(0)
        }
    }

    async function update_match_kyous_for_image(): Promise<void> {
        // 全件を舐めて行に詰め直すので、生の配列を読む。
        // deepなref配下のリアクティブProxy越しに読むと1要素ごとにProxyを確保する。
        // 詰める先(match_kyous_for_image)はリアクティブのままなので描画は追随する。
        const raw_matched_kyous = props.matched_kyous ? toRaw(props.matched_kyous) : null
        const column_number = props.application_config.rykv_image_list_column_number.valueOf()
        const match_kyous_for_image_result = new Array<Array<Kyou>>()
        for (let i = 0; raw_matched_kyous && i < raw_matched_kyous.length;) {
            const kyou_row_list = new Array<Kyou>()
            for (let j = 0; j < column_number; j++) {
                if (i < raw_matched_kyous.length) {
                    kyou_row_list.push(raw_matched_kyous[i])
                    i++
                }
            }
            match_kyous_for_image_result.push(kyou_row_list)
        }
        // 一度空にするとv-virtual-scrollのコンテンツ高さが0になりscrollTopがリセットされるため、
        // 空にせず変更のあった行だけ置き換える
        for (let i = 0; i < match_kyous_for_image_result.length; i++) {
            const current_row = match_kyous_for_image.value[i]
            const new_row = match_kyous_for_image_result[i]
            const is_same_row = current_row && current_row.length === new_row.length
                && current_row.every((kyou, j) => kyou === new_row[j])
            if (!is_same_row) {
                match_kyous_for_image.value[i] = new_row
            }
        }
        if (match_kyous_for_image.value.length > match_kyous_for_image_result.length) {
            match_kyous_for_image.value.splice(match_kyous_for_image_result.length)
        }
    }

    // ── Exposed methods ──

    // scroll_toのリトライ世代。新しいscroll_to呼び出し・unmountで古いリトライチェーンを破棄する。
    // 世代なしの自己再帰だと、0件のまま終わった列(scrollHeight=0)への呼び出しが
    // 50ms間隔の強制レイアウト付きで永久に残り、列操作のたびに増殖してレンダラを飽和させる
    let scroll_to_epoch = 0
    const scroll_to_max_retry_count = 40 // 50ms x 40 = 約2秒でリトライを打ち切る

    onUnmounted(() => { scroll_to_epoch++ })

    async function scroll_to(scroll_top: number): Promise<void> {
        const epoch = ++scroll_to_epoch
        return nextTick(async () => {
            try_scroll_to(scroll_top, epoch, 0)
        })
    }

    function try_scroll_to(scroll_top: number, epoch: number, retry_count: number): void {
        if (epoch !== scroll_to_epoch) {
            return
        }
        const target_element_id = props.query.query_id.concat(props.query.is_image_only ? "_kyou_image_list_view" : "_kyou_list_view")
        const kyou_list_view_element = document.getElementById(target_element_id)
        const virtual_scroll_container = kyou_list_view_element?.querySelector(".v-virtual-scroll__container")
        const scroll_height = virtual_scroll_container?.scrollHeight ?? kyou_list_view_element?.scrollHeight
        // 要素がまだ無い/描画前(高さ0)/目標に届かない間は少し待って引き直す。
        // ただし上限まで。以前は打ち切りが無く、別条件の再検索で件数が減った列へ
        // 保存済みスクロール位置を復元するケースなどで成立し得ない条件を永久に待っていた
        if (!kyou_list_view_element || !scroll_height || scroll_height < scroll_top) {
            if (retry_count >= scroll_to_max_retry_count) {
                if (kyou_list_view_element) {
                    // 要素はあるが高さが目標に届かないまま打ち切り。
                    // scrollTopは範囲外ならブラウザがクランプするのでそのまま代入する
                    kyou_list_view_element.scrollTop = (scroll_top)
                }
                return
            }
            nextTick(async () => { // nextTickじゃ動かんかったのでsleepで対応
                await sleep(50)
                try_scroll_to(scroll_top, epoch, retry_count + 1)
            })
            return
        }
        kyou_list_view_element.scrollTop = (scroll_top)
    }

    async function scroll_to_kyou(kyou: Kyou): Promise<boolean> {
        // 行クリックのたびに全件を舐めるので、生の配列を読む。
        // deepなref配下のリアクティブProxy越しに読むと1要素ごとに track と toReactive が走り、
        // 要素ぶんのProxyを確保する(30万件の列では効く)。読み取りだけなので意味論は変わらない。
        //
        // 二分探索にはしない: 単調なのはrykv(related_time降順)だけで、mi列は
        // 射影時刻の昇順＋未設定が末尾(compare_kyou_for_query)なので分岐が要り、
        // 間違えると黙って違う場所へスクロールする。
        const raw_matched_kyous = toRaw(props.matched_kyous)
        let index = -1;
        for (let i = 0; i < raw_matched_kyous.length; i++) {
            const kyou_in_list = raw_matched_kyous[i]
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
        // scroll_to_kyou と同じ理由で生の配列を読む(こちらはフォーカス時刻が変わるたびに走る)
        const raw_matched_kyous = toRaw(props.matched_kyous)
        let index = -1;
        for (let i = 0; i < raw_matched_kyous.length; i++) {
            const kyou = raw_matched_kyous[i]
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
        if (loading) {
            // 再検索中は「読み込み済み」を倒す。列は検索で再マウントされないので、
            // ここで戻さないと空にした直後のリストが「該当なし」と誤表示される
            has_loaded.value = false
        } else {
            has_loaded.value = true
        }
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
        has_loaded,

        // Computed
        kyou_height_px,
        footer_height,
        footer_class,

        // Business logic
        should_show_related_time,

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
