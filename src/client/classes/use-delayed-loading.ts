'use strict'

import { getCurrentScope, onScopeDispose, ref, watch, type ComputedRef, type Ref } from 'vue'

/**
 * 読み込み中表示を出すまでの待ち時間(ms)。
 *
 * Kyouはどの種別でも表示のたびにAPIを1回叩くので、一覧では数十行が同時に読み込み中になる。
 * ローカルサーバだと数十msで返るため、待たずに出すとスクロールのたびにスピナーが一斉に明滅する。
 * この時間を超えたときだけ出すことで、速く終わる読み込みは従来どおり何も出さずに済ませる。
 */
export const LOADING_INDICATOR_DELAY_MS = 200

/**
 * 読み込み中フラグが一定時間続いたときだけ立つフラグを返す。
 *
 * 速く終わった読み込みでは立たないので、インジケータの明滅を防げる。
 * タイマーはスコープ破棄時に必ず片付ける（仮想スクロールの行使い回しで大量に生成・破棄されるため）。
 */
export function useDelayedLoading(
    is_loading: Ref<boolean> | ComputedRef<boolean>,
    delay_ms: number = LOADING_INDICATOR_DELAY_MS,
): Ref<boolean> {
    const is_delay_elapsed: Ref<boolean> = ref(false)
    let timer_id: ReturnType<typeof setTimeout> | null = null

    function clear_timer(): void {
        if (timer_id !== null) {
            clearTimeout(timer_id)
            timer_id = null
        }
    }

    // immediate: trueで初期状態も拾う
    watch(is_loading, (loading) => {
        clear_timer()
        if (!loading) {
            is_delay_elapsed.value = false
            return
        }
        timer_id = setTimeout(() => {
            is_delay_elapsed.value = true
            timer_id = null
        }, delay_ms)
    }, { immediate: true })

    // onUnmountedではなくonScopeDisposeを使う。
    // コンポーネント外（ユニットテストなど）でcomposableを直接呼んでも警告が出ないよう、
    // スコープが無いときは登録自体を飛ばす
    if (getCurrentScope()) {
        onScopeDispose(clear_timer)
    }

    return is_delay_elapsed
}
