import { onBeforeUnmount, onMounted, type Ref, watch } from "vue"

/**
 * Dialog history stack manager (lightweight, router-friendly)
 * + resetDialogHistory()
 * + Back closes topmost dialog first when dialogs are open.
 *
 * 案C: ×ボタン/Escape も closeDialogViaHistory() で「ブラウザバック相当」に一本化。
 * - closeDialogViaHistory(ref) は history.go(-1) を発行し、popstate ハンドラが
 *   ref を false にする。ブラウザバックと完全に同じ経路になるため、
 *   バックスタックに使用済みエントリが残らない。
 * - onClosed オプション: 閉じた原因を問わず(バック/×/Escape/リセット)ちょうど1回
 *   呼ばれる。'closed' emit はここに一本化する(hide() 内での emit は撤去)。
 *   これにより「emit → 親が unmount → post-flush watcher が破棄され巻き戻し漏れ」
 *   の競合も構造的に消える。
 * - 注意: iframe 内ナビゲーションで joint history entry を作るダイアログ
 *   (help/tutorial/plugin-config)は closeDialogViaHistory を使ってはならない。
 *   go(-1) が iframe 側の履歴を巻いてしまい top-level popstate が発火しないため。
 *   それらは従来通り ref を直接 false にする(プログラム的クローズ分岐が
 *   unmount で iframe が消えた後に巻き戻すので、joint entry ごと消えて安全)。
 *
 * Extra:
 * - Escape closes ONLY the topmost dialog (no global "全部閉じる" 問題を回避)
 *
 * Notes:
 * - Forward into dialog states while stack is empty is also blocked.
 */

const MARK = "__gkillDlg"
const DEPTH = "__gkillDlgDepth"

type AnyObj = Record<string, unknown>
function isObj(v: unknown): v is AnyObj {
  return v !== null && typeof v === "object"
}
function isDialogState(state: unknown): boolean {
  return isObj(state) && state[MARK] === true && typeof state[DEPTH] === "number"
}
function stripDialogKeys(state: unknown): unknown {
  if (!isObj(state)) return state
  if (!(MARK in state) && !(DEPTH in state)) return state
  const { [MARK]: _m, [DEPTH]: _d, ...rest } = state
  return rest
}
function withDialogMarkers(base: unknown, depth: number): AnyObj {
  const b: AnyObj = isObj(base) ? (base as AnyObj) : {}
  return { ...b, [MARK]: true, [DEPTH]: depth }
}

// --- Global stack (module singleton) ---
type Entry = { id: string; dialog: Ref<boolean> }
const stack: Entry[] = []

export type UseDialogHistoryStackOptions = {
  // 閉じたとき(バック/×/Escape/リセットを問わず)にちょうど1回呼ばれる。
  // 'closed' emit の一本化用。unmount による強制クローズでは呼ばれない。
  onClosed?: () => void
}
const onClosedMap = new WeakMap<object, () => void>()

// --- Pending open tracking (setTimeout-based push delay) ---
// By deferring history.pushState to a macro-task (setTimeout 0), any iframe
// navigations triggered during the same render cycle (also macro-tasks, but
// queued earlier) complete first at the pre-dialog top-level state.
// This prevents their joint session history entries from landing after the
// dialog's push entry, which would cause popstate to not fire on back and
// require 2 back presses to close the dialog.
let openSeqCounter = 0
const pendingOpenSeqMap = new WeakMap<object, number>()

function scheduleOpen(id: string, dialog: Ref<boolean>, refObj: object): void {
  const seq = ++openSeqCounter
  pendingOpenSeqMap.set(refObj, seq)
  setTimeout(() => {
    if (pendingOpenSeqMap.get(refObj) !== seq) return
    pendingOpenSeqMap.delete(refObj)
    if (!dialog.value) return
    if (navInFlight()) {
      queueOpen(id, dialog)
      return
    }
    const existingIdx = stack.findIndex((e) => e.id === id)
    if (existingIdx >= 0) {
      const [e] = stack.splice(existingIdx, 1)
      stack.push(e)
    } else {
      stack.push({ id, dialog })
    }
    pushDialogHistory(stack.length)
  }, 0)
}

// Identity helpers
const refIdMap = new WeakMap<object, string>()
let refIdSeq = 0
function getRefId(r: object): string {
  const existing = refIdMap.get(r)
  if (existing) return existing
  const id = `dlg_${(++refIdSeq).toString(16)}`
  refIdMap.set(r, id)
  return id
}

// Prevent multi-registration for same Ref<boolean>
const watchedRefs = new WeakSet<object>()

// When we close a dialog because of popstate, watcher should NOT call history.go again.
const closingFromPop = new WeakSet<object>()
// When we close because of resetDialogHistory, watcher should NOT call history.go again.
const closingFromReset = new WeakSet<object>()

// --- Race protection: queue opens while a history navigation is pending ---
let pendingNav = 0 // 自前の巻き戻し。着弾する popstate を握りつぶす数
const queuedOpens: Array<{ id: string; dialog: Ref<boolean> }> = []

// --- 案C: history 駆動クローズの管理 ---
// closeDialogViaHistory() が発行した go(-1) の飛行数。着弾 popstate は
// (pendingNav と違い)通常の back と同じように処理される。
let pendingCloseNav = 0
// 閉じる対象として指定されたダイアログ (FIFO)。popstate 側で最優先で閉じる。
// これにより「最上位以外の×」も正しいダイアログが閉じる。
const pendingCloseTargets: object[] = []
// closeDialogViaHistory の完了待ち Promise
const closeWaiters = new Map<object, { promise: Promise<void>; resolve: () => void }>()

function waiterFor(refObj: object): Promise<void> {
  const existing = closeWaiters.get(refObj)
  if (existing) return existing.promise
  let resolve!: () => void
  const promise = new Promise<void>((r) => {
    resolve = r
  })
  closeWaiters.set(refObj, { promise, resolve })
  return promise
}
function resolveCloseWaiter(refObj: object): void {
  const w = closeWaiters.get(refObj)
  if (!w) return
  closeWaiters.delete(refObj)
  w.resolve()
}

function navInFlight(): boolean {
  return pendingNav > 0 || pendingCloseNav > 0
}

function queueOpen(id: string, dialog: Ref<boolean>) {
  const idx = queuedOpens.findIndex((x) => x.id === id)
  if (idx >= 0) queuedOpens.splice(idx, 1)
  queuedOpens.push({ id, dialog })
}

function pushDialogHistory(depth: number) {
  const base = history.state
  history.pushState(withDialogMarkers(base, depth), "")
}

function flushQueuedOpens() {
  if (navInFlight() || queuedOpens.length === 0) return
  const items = queuedOpens.splice(0, queuedOpens.length)

  for (const it of items) {
    if (it.dialog.value !== true) continue

    const existingIdx = stack.findIndex((e) => e.id === it.id)
    if (existingIdx >= 0) {
      const [e] = stack.splice(existingIdx, 1)
      stack.push(e)
    } else {
      stack.push({ id: it.id, dialog: it.dialog })
    }

    pushDialogHistory(stack.length)
  }
}

function clearDialogKeysFromCurrentState() {
  if (stack.length !== 0) return
  const cleaned = stripDialogKeys(history.state)
  if (cleaned === history.state) return
  history.replaceState(cleaned, "")
}

// --- reset waiters ---
let resetWaiters: Array<() => void> = []
function resolveResetWaiters() {
  if (resetWaiters.length === 0) return
  const ws = resetWaiters
  resetWaiters = []
  for (const w of ws) w()
}

// ナビゲーション完了後の共通後始末
function afterNavSettled(): void {
  if (navInFlight()) return
  if (stack.length === 0) clearDialogKeysFromCurrentState()
  flushQueuedOpens()
  resolveResetWaiters()
}

/**
 * Close all dialogs and rewind browser history by the dialog depth.
 */
export function resetDialogHistory(): Promise<void> {
  if (!navInFlight() && stack.length === 0) return Promise.resolve()

  const depth = stack.length
  return new Promise<void>((resolve) => {
    resetWaiters.push(resolve)

    if (depth <= 0) {
      if (!navInFlight()) resolveResetWaiters()
      return
    }

    const entries = stack.slice()
    // Clear stack immediately to avoid double-close by popstate order.
    stack.length = 0

    for (let i = entries.length - 1; i >= 0; i--) {
      const refObj = entries[i].dialog as unknown as object
      closingFromReset.add(refObj)
      entries[i].dialog.value = false
    }

    // 飛行中の history 駆動クローズ traversal は既に1エントリずつ消費する。
    // その分を差し引いて巻き戻し、着弾する popstate はすべて握りつぶす。
    // 注意: popstate は「エントリ数」ではなく「トラバーサル数」だけ発火する。
    // go(-N) は N エントリ戻っても popstate 1回なので、握りつぶし予約も
    // トラバーサル単位で数える (エントリ数で数えると N>=2 で pendingNav が
    // 詰まり、resetDialogHistory の Promise が永遠に resolve しなくなる)。
    const inFlight = pendingCloseNav
    pendingCloseNav = 0
    pendingCloseTargets.length = 0
    const goDelta = Math.max(0, depth - inFlight)
    pendingNav += inFlight + (goDelta > 0 ? 1 : 0)
    if (goDelta > 0) history.go(-goDelta)
  })
}

/**
 * ダイアログを「ブラウザバック相当」で閉じる (案C)。
 * 履歴エントリを1つ消費する go(-1) を発行し、popstate ハンドラが ref を
 * false にする。ブラウザバック(①)と×ボタン(②)が完全に同一経路になる。
 * - 履歴に積まれる前(pending/queued open)なら直接 false にする。
 * - useDialogHistoryStack 未登録の ref も直接 false にする。
 * - 戻り値は実際に閉じ終わったとき(watcher 処理後)に resolve する。
 * - iframe 内ナビゲーションで joint history entry を作るダイアログには使用禁止
 *   (ファイル先頭コメント参照)。
 */
export function closeDialogViaHistory(dialog: Ref<boolean>): Promise<void> {
  const refObj = dialog as unknown as object
  if (!dialog.value) return Promise.resolve()

  if (!watchedRefs.has(refObj)) {
    dialog.value = false
    return Promise.resolve()
  }

  const id = getRefId(refObj)
  const inStack = stack.findIndex((e) => e.id === id) >= 0
  if (!inStack) {
    // まだ pushState されていない: 直接閉じる (watcher 側でキャンセル処理される)
    const p = waiterFor(refObj)
    dialog.value = false
    return p
  }

  if (pendingCloseTargets.indexOf(refObj) >= 0) return waiterFor(refObj)

  const p = waiterFor(refObj)
  pendingCloseTargets.push(refObj)
  pendingCloseNav++
  history.go(-1)
  return p
}

// Public helper: close only the topmost dialog (history 駆動)。
// 既にクローズ要求済みのものはスキップするので、連打で順に閉じられる。
export function closeTopDialog(): boolean {
  for (let i = stack.length - 1; i >= 0; i--) {
    const refObj = stack[i].dialog as unknown as object
    if (pendingCloseTargets.indexOf(refObj) >= 0) continue
    closeDialogViaHistory(stack[i].dialog)
    return true
  }
  return false
}

// --- Back handling ---
// When no dialogs are open, do not consume back navigation.
// This allows normal browser/PWA behavior (including app close in standalone mode).
const backOnlyEnabled = false
let backOnlyBouncePending = 0 // prevents infinite loops when we call history.go(1)

// --- popstate handling ---
let popListenerInstalled = false
function ensurePopListenerInstalled() {
  if (popListenerInstalled) return
  popListenerInstalled = true

  // リロード直後など、ダイアログが無いのに現在エントリにマーカーが残っている
  // 場合を無害化しておく (残っていると×クローズの深さ判定が狂う)
  if (stack.length === 0) clearDialogKeysFromCurrentState()

  window.addEventListener(
    "popstate",
    (e) => {
      // A) If this popstate was caused by our own history.go(+/-), swallow it.
      if (pendingNav > 0) {
        pendingNav--
        if (pendingNav === 0) afterNavSettled()
        return
      }

      // 案C: closeDialogViaHistory の traversal が着弾。
      // 以降は通常のブラウザバックと同じ扱いで処理する。
      if (pendingCloseNav > 0) pendingCloseNav--

      // B) Back-only bounce (we called history.go(1) to cancel a back)
      if (backOnlyBouncePending > 0) {
        backOnlyBouncePending = 0
        return
      }

      // C) Forward into dialog state while stack is empty: strip markers
      if (stack.length === 0 && isDialogState((e as PopStateEvent).state)) {
        history.replaceState(stripDialogKeys((e as PopStateEvent).state), "")
        afterNavSettled()
        return
      }

      // D) If any dialog is open
      if (stack.length > 0) {
        const newDepth = isDialogState((e as PopStateEvent).state)
          ? ((e as PopStateEvent).state as AnyObj)[DEPTH] as number
          : 0

        // Forward: depth in new state >= current stack → don't close
        if (newDepth >= stack.length) {
          history.replaceState(
            withDialogMarkers(stripDialogKeys((e as PopStateEvent).state), stack.length),
            ""
          )
          afterNavSettled()
          return
        }

        // Back: 深さの差分だけ閉じる (長押しジャンプで複数エントリ戻った場合も
        // 追従する)。closeDialogViaHistory のターゲットがあれば最優先で閉じる。
        try { (e as PopStateEvent).stopImmediatePropagation?.() } catch { /* ignore */ }

        let count = stack.length - newDepth
        while (count > 0 && stack.length > 0) {
          let idx = -1
          while (pendingCloseTargets.length > 0 && idx < 0) {
            const t = pendingCloseTargets.shift() as object
            idx = stack.findIndex((en) => (en.dialog as unknown as object) === t)
          }
          if (idx < 0) idx = stack.length - 1

          const [entry] = stack.splice(idx, 1)
          const refObj = entry.dialog as unknown as object
          if (entry.dialog.value === true) {
            closingFromPop.add(refObj)
            entry.dialog.value = false
          } else {
            // 既に閉じている残骸だった場合は待ちだけ解決する
            resolveCloseWaiter(refObj)
          }
          count--
        }
        if (stack.length === 0) clearDialogKeysFromCurrentState()
        afterNavSettled()
        return
      }

      // E) Dialog-only mode: when stack is empty, block back navigation.
      //    This keeps the user on the current route; back is reserved for dialogs.
      if (backOnlyEnabled && stack.length === 0) {
        try {
          (e as PopStateEvent).stopImmediatePropagation?.()
        } catch {
          // ignore
        }

        // Cancel this back by going forward one step.
        // Mark so we don't loop.
        backOnlyBouncePending = 1
        history.go(1)
        return
      }

      afterNavSettled()
    },
    { capture: true } as AddEventListenerOptions,
  )
}

// --- Escape handling (close only topmost) ---
let escListenerInstalled = false
function ensureEscListenerInstalled() {
  if (escListenerInstalled) return
  escListenerInstalled = true

  window.addEventListener(
    "keydown",
    (e: KeyboardEvent) => {
      if (e.key !== "Escape") return
      if (e.repeat) return
      if (stack.length === 0) return

      // popstate の処理中にさらに閉じるとややこしいので避ける
      if (pendingNav > 0) return

      e.preventDefault()
      e.stopPropagation()

      // 1回の ESC で 1つだけ閉じる (×ボタンと同じく history 駆動)
      closeTopDialog()
    },
    { capture: true },
  )
}

// --- Core hook ---
export function useDialogHistoryStack(
  dialog: Ref<boolean>,
  options?: UseDialogHistoryStackOptions,
): void {
  const refObj = dialog as unknown as object
  const id = getRefId(refObj)

  if (options?.onClosed) onClosedMap.set(refObj, options.onClosed)

  if (watchedRefs.has(refObj)) {
    ensurePopListenerInstalled()
    ensureEscListenerInstalled()
    return
  }
  watchedRefs.add(refObj)
  ensurePopListenerInstalled()
  ensureEscListenerInstalled()

  const stop = watch(
    dialog,
    (open) => {
      if (open) {
        if (navInFlight()) {
          queueOpen(id, dialog)
          return
        }
        scheduleOpen(id, dialog, refObj)
        return
      }

      // ---- close ----
      if (closingFromPop.has(refObj)) {
        // close (pop): popstate ハンドラ側で stack から除去済みのことが多い
        closingFromPop.delete(refObj)
        const idx = stack.findIndex((en) => en.id === id)
        if (idx >= 0) stack.splice(idx, 1)
        if (stack.length === 0) clearDialogKeysFromCurrentState()
      } else if (closingFromReset.has(refObj)) {
        // close (reset)
        closingFromReset.delete(refObj)
        if (stack.length === 0) clearDialogKeysFromCurrentState()
      } else if (pendingOpenSeqMap.has(refObj)) {
        // Cancel pending open if dialog closed before setTimeout fired
        pendingOpenSeqMap.delete(refObj)
        if (stack.length === 0) clearDialogKeysFromCurrentState()
      } else {
        const idx = stack.findIndex((en) => en.id === id)
        if (idx >= 0) {
          // Programmatic close (ref 直接書き換え)。該当エントリのみ除去して
          // 1エントリ巻き戻す (上に乗っているダイアログはそのまま維持される)。
          stack.splice(idx, 1)

          const targetIdx = pendingCloseTargets.indexOf(refObj)
          if (targetIdx >= 0) {
            // closeDialogViaHistory の traversal が飛行中に直接 false にされた:
            // 二重に戻らず、着弾する popstate を握りつぶしに変換する
            pendingCloseTargets.splice(targetIdx, 1)
            pendingCloseNav = Math.max(0, pendingCloseNav - 1)
            pendingNav += 1
          } else {
            pendingNav += 1
            history.go(-1)
          }
        } else if (stack.length === 0) {
          clearDialogKeysFromCurrentState()
        }
      }

      resolveCloseWaiter(refObj)
      onClosedMap.get(refObj)?.()
    },
    { flush: "post" },
  )

  onBeforeUnmount(() => {
    stop()
    watchedRefs.delete(refObj)
    closingFromPop.delete(refObj)
    closingFromReset.delete(refObj)
    pendingOpenSeqMap.delete(refObj)
    onClosedMap.delete(refObj)

    if (dialog.value === true) dialog.value = false

    const idx = stack.findIndex((en) => en.id === id)
    if (idx >= 0) stack.splice(idx, 1)

    // 飛行中の close traversal は unmount 後に着弾するため握りつぶしに変換
    const targetIdx = pendingCloseTargets.indexOf(refObj)
    if (targetIdx >= 0) {
      pendingCloseTargets.splice(targetIdx, 1)
      pendingCloseNav = Math.max(0, pendingCloseNav - 1)
      pendingNav += 1
    }

    if (stack.length === 0) clearDialogKeysFromCurrentState()
    resolveCloseWaiter(refObj)
  })

  onMounted(() => {
    if (dialog.value === true) {
      if (navInFlight()) {
        queueOpen(id, dialog)
        return
      }
      scheduleOpen(id, dialog, refObj)
    }
  })
}
