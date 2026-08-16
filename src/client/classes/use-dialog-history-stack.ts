import { getCurrentInstance, onBeforeUnmount, onMounted, type Ref, watch } from "vue"

/**
 * Dialog history stack manager (lightweight, router-friendly)
 * + reset_dialog_history()
 * + Back closes topmost dialog first when dialogs are open.
 *
 * 案C: ×ボタン/Escape も close_dialog_via_history() で「ブラウザバック相当」に一本化。
 * - close_dialog_via_history(ref) は history.go(-1) を発行し、popstate ハンドラが
 *   ref を false にする。ブラウザバックと完全に同じ経路になるため、
 *   バックスタックに使用済みエントリが残らない。
 * - onClosed オプション: 閉じた原因を問わず(バック/×/Escape/リセット)ちょうど1回
 *   呼ばれる。'closed' emit はここに一本化する(hide() 内での emit は撤去)。
 *   これにより「emit → 親が unmount → post-flush watcher が破棄され巻き戻し漏れ」
 *   の競合も構造的に消える。
 * - 注意: iframe 内ナビゲーションで joint history entry を作るダイアログ
 *   (help/tutorial/plugin-config)は close_dialog_via_history を使ってはならない。
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
function is_obj(v: unknown): v is AnyObj {
  return v !== null && typeof v === "object"
}
function is_dialog_state(state: unknown): boolean {
  return is_obj(state) && state[MARK] === true && typeof state[DEPTH] === "number"
}
function strip_dialog_keys(state: unknown): unknown {
  if (!is_obj(state)) return state
  if (!(MARK in state) && !(DEPTH in state)) return state
  const { [MARK]: _m, [DEPTH]: _d, ...rest } = state
  return rest
}
function with_dialog_markers(base: unknown, depth: number): AnyObj {
  const b: AnyObj = is_obj(base) ? (base as AnyObj) : {}
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
let open_seq_counter = 0
const pending_open_seq_map = new WeakMap<object, number>()

function schedule_open(id: string, dialog: Ref<boolean>, ref_obj: object): void {
  const seq = ++open_seq_counter
  pending_open_seq_map.set(ref_obj, seq)
  setTimeout(() => {
    if (pending_open_seq_map.get(ref_obj) !== seq) return
    pending_open_seq_map.delete(ref_obj)
    if (!dialog.value) return
    if (nav_in_flight()) {
      queue_open(id, dialog)
      return
    }
    const existing_idx = stack.findIndex((e) => e.id === id)
    if (existing_idx >= 0) {
      const [e] = stack.splice(existing_idx, 1)
      stack.push(e)
    } else {
      stack.push({ id, dialog })
    }
    push_dialog_history(stack.length)
  }, 0)
}

// Identity helpers
const ref_id_map = new WeakMap<object, string>()
let ref_id_seq = 0
function get_ref_id(r: object): string {
  const existing = ref_id_map.get(r)
  if (existing) return existing
  const id = `dlg_${(++ref_id_seq).toString(16)}`
  ref_id_map.set(r, id)
  return id
}

// Prevent multi-registration for same Ref<boolean>
const watched_refs = new WeakSet<object>()

// --- 最前面のダイアログを閉じるための対応付け ---
//
// バックと Escape は `stack` の末尾を閉じる。ダイアログはクリックで前面へ出せる
// （use-floating-dialog.ts の z_order）ので、「積んだ順の末尾」と
// 「見た目の最前面」がずれる。ずれたままだと、奥のダイアログをバックで閉じてしまう。
//
// `useDialogHistoryStack` と `useFloatingDialog` は同じコンポーネントの setup で
// 呼ばれるので、コンポーネントインスタンスを鍵にして両者を結ぶ。
const ref_owner_map = new WeakMap<object, object>()

/**
 * 前面へ出たダイアログ（とその子孫）の履歴エントリを末尾へ移す。
 *
 * 引数は奥から手前の順。相対順を保ったまま末尾へ積み直すので、
 * 親を前面へ出しても、その親から開いた確認ダイアログは親より手前に残る。
 */
export function raise_dialog_history_entries(owners: ReadonlyArray<object>): void {
  if (owners.length === 0) return

  const moved: Entry[] = []
  for (const owner of owners) {
    const entry = stack.find((e) => ref_owner_map.get(e.dialog as unknown as object) === owner)
    if (entry && !moved.includes(entry)) moved.push(entry)
  }
  if (moved.length === 0) return

  // すでに同じ並びで末尾に居るなら触らない
  const tail_start = stack.length - moved.length
  let already_top = true
  for (let i = 0; i < moved.length; i++) {
    if (stack[tail_start + i] !== moved[i]) {
      already_top = false
      break
    }
  }
  if (already_top) return

  for (const entry of moved) {
    stack.splice(stack.indexOf(entry), 1)
  }
  stack.push(...moved)
}

// When we close a dialog because of popstate, watcher should NOT call history.go again.
const closing_from_pop = new WeakSet<object>()
// When we close because of reset_dialog_history, watcher should NOT call history.go again.
const closing_from_reset = new WeakSet<object>()

// --- Race protection: queue opens while a history navigation is pending ---
let pending_nav = 0 // 自前の巻き戻し。着弾する popstate を握りつぶす数
const queued_opens: Array<{ id: string; dialog: Ref<boolean> }> = []

// --- 案C: history 駆動クローズの管理 ---
// close_dialog_via_history() が発行した go(-1) の飛行数。着弾 popstate は
// (pending_nav と違い)通常の back と同じように処理される。
let pending_close_nav = 0
// 閉じる対象として指定されたダイアログ (FIFO)。popstate 側で最優先で閉じる。
// これにより「最上位以外の×」も正しいダイアログが閉じる。
const pending_close_targets: object[] = []
// close_dialog_via_history の完了待ち Promise
const close_waiters = new Map<object, { promise: Promise<void>; resolve: () => void }>()

function waiter_for(ref_obj: object): Promise<void> {
  const existing = close_waiters.get(ref_obj)
  if (existing) return existing.promise
  let resolve!: () => void
  const promise = new Promise<void>((r) => {
    resolve = r
  })
  close_waiters.set(ref_obj, { promise, resolve })
  return promise
}
function resolve_close_waiter(ref_obj: object): void {
  const w = close_waiters.get(ref_obj)
  if (!w) return
  close_waiters.delete(ref_obj)
  w.resolve()
}

function nav_in_flight(): boolean {
  return pending_nav > 0 || pending_close_nav > 0
}

function queue_open(id: string, dialog: Ref<boolean>) {
  const idx = queued_opens.findIndex((x) => x.id === id)
  if (idx >= 0) queued_opens.splice(idx, 1)
  queued_opens.push({ id, dialog })
}

function push_dialog_history(depth: number) {
  const base = history.state
  history.pushState(with_dialog_markers(base, depth), "")
}

function flush_queued_opens() {
  if (nav_in_flight() || queued_opens.length === 0) return
  const items = queued_opens.splice(0, queued_opens.length)

  for (const it of items) {
    if (it.dialog.value !== true) continue

    const existing_idx = stack.findIndex((e) => e.id === it.id)
    if (existing_idx >= 0) {
      const [e] = stack.splice(existing_idx, 1)
      stack.push(e)
    } else {
      stack.push({ id: it.id, dialog: it.dialog })
    }

    push_dialog_history(stack.length)
  }
}

function clear_dialog_keys_from_current_state() {
  if (stack.length !== 0) return
  const cleaned = strip_dialog_keys(history.state)
  if (cleaned === history.state) return
  history.replaceState(cleaned, "")
}

// --- reset waiters ---
let reset_waiters: Array<() => void> = []
function resolve_reset_waiters() {
  if (reset_waiters.length === 0) return
  const ws = reset_waiters
  reset_waiters = []
  for (const w of ws) w()
}

// ナビゲーション完了後の共通後始末
function after_nav_settled(): void {
  if (nav_in_flight()) return
  if (stack.length === 0) clear_dialog_keys_from_current_state()
  flush_queued_opens()
  resolve_reset_waiters()
}

/**
 * Close all dialogs and rewind browser history by the dialog depth.
 */
export function reset_dialog_history(): Promise<void> {
  if (!nav_in_flight() && stack.length === 0) return Promise.resolve()

  const depth = stack.length
  return new Promise<void>((resolve) => {
    reset_waiters.push(resolve)

    if (depth <= 0) {
      if (!nav_in_flight()) resolve_reset_waiters()
      return
    }

    const entries = stack.slice()
    // Clear stack immediately to avoid double-close by popstate order.
    stack.length = 0

    for (let i = entries.length - 1; i >= 0; i--) {
      const ref_obj = entries[i].dialog as unknown as object
      closing_from_reset.add(ref_obj)
      entries[i].dialog.value = false
    }

    // 飛行中の history 駆動クローズ traversal は既に1エントリずつ消費する。
    // その分を差し引いて巻き戻し、着弾する popstate はすべて握りつぶす。
    // 注意: popstate は「エントリ数」ではなく「トラバーサル数」だけ発火する。
    // go(-N) は N エントリ戻っても popstate 1回なので、握りつぶし予約も
    // トラバーサル単位で数える (エントリ数で数えると N>=2 で pending_nav が
    // 詰まり、reset_dialog_history の Promise が永遠に resolve しなくなる)。
    const in_flight = pending_close_nav
    pending_close_nav = 0
    pending_close_targets.length = 0
    const go_delta = Math.max(0, depth - in_flight)
    pending_nav += in_flight + (go_delta > 0 ? 1 : 0)
    if (go_delta > 0) history.go(-go_delta)
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
export function close_dialog_via_history(dialog: Ref<boolean>): Promise<void> {
  const ref_obj = dialog as unknown as object
  if (!dialog.value) return Promise.resolve()

  if (!watched_refs.has(ref_obj)) {
    dialog.value = false
    return Promise.resolve()
  }

  const id = get_ref_id(ref_obj)
  const in_stack = stack.findIndex((e) => e.id === id) >= 0
  if (!in_stack) {
    // まだ pushState されていない: 直接閉じる (watcher 側でキャンセル処理される)
    const p = waiter_for(ref_obj)
    dialog.value = false
    return p
  }

  if (pending_close_targets.indexOf(ref_obj) >= 0) return waiter_for(ref_obj)

  const p = waiter_for(ref_obj)
  pending_close_targets.push(ref_obj)
  pending_close_nav++
  history.go(-1)
  return p
}

// Public helper: close only the topmost dialog (history 駆動)。
// 既にクローズ要求済みのものはスキップするので、連打で順に閉じられる。
export function close_top_dialog(): boolean {
  for (let i = stack.length - 1; i >= 0; i--) {
    const ref_obj = stack[i].dialog as unknown as object
    if (pending_close_targets.indexOf(ref_obj) >= 0) continue
    close_dialog_via_history(stack[i].dialog)
    return true
  }
  return false
}

// --- Back handling ---
// When no dialogs are open, do not consume back navigation.
// This allows normal browser/PWA behavior (including app close in standalone mode).
const back_only_enabled = false
let back_only_bounce_pending = 0 // prevents infinite loops when we call history.go(1)

// --- popstate handling ---
let pop_listener_installed = false
function ensure_pop_listener_installed() {
  if (pop_listener_installed) return
  pop_listener_installed = true

  // リロード直後など、ダイアログが無いのに現在エントリにマーカーが残っている
  // 場合を無害化しておく (残っていると×クローズの深さ判定が狂う)
  if (stack.length === 0) clear_dialog_keys_from_current_state()

  window.addEventListener(
    "popstate",
    (e) => {
      // A) If this popstate was caused by our own history.go(+/-), swallow it.
      if (pending_nav > 0) {
        pending_nav--
        if (pending_nav === 0) after_nav_settled()
        return
      }

      // 案C: close_dialog_via_history の traversal が着弾。
      // 以降は通常のブラウザバックと同じ扱いで処理する。
      if (pending_close_nav > 0) pending_close_nav--

      // B) Back-only bounce (we called history.go(1) to cancel a back)
      if (back_only_bounce_pending > 0) {
        back_only_bounce_pending = 0
        return
      }

      // C) Forward into dialog state while stack is empty: strip markers
      if (stack.length === 0 && is_dialog_state((e as PopStateEvent).state)) {
        history.replaceState(strip_dialog_keys((e as PopStateEvent).state), "")
        after_nav_settled()
        return
      }

      // D) If any dialog is open
      if (stack.length > 0) {
        const new_depth = is_dialog_state((e as PopStateEvent).state)
          ? ((e as PopStateEvent).state as AnyObj)[DEPTH] as number
          : 0

        // Forward: depth in new state >= current stack → don't close
        if (new_depth >= stack.length) {
          history.replaceState(
            with_dialog_markers(strip_dialog_keys((e as PopStateEvent).state), stack.length),
            ""
          )
          after_nav_settled()
          return
        }

        // Back: 深さの差分だけ閉じる (長押しジャンプで複数エントリ戻った場合も
        // 追従する)。close_dialog_via_history のターゲットがあれば最優先で閉じる。
        try { (e as PopStateEvent).stopImmediatePropagation?.() } catch { /* ignore */ }

        let count = stack.length - new_depth
        while (count > 0 && stack.length > 0) {
          let idx = -1
          while (pending_close_targets.length > 0 && idx < 0) {
            const t = pending_close_targets.shift() as object
            idx = stack.findIndex((en) => (en.dialog as unknown as object) === t)
          }
          if (idx < 0) idx = stack.length - 1

          const [entry] = stack.splice(idx, 1)
          const ref_obj = entry.dialog as unknown as object
          if (entry.dialog.value === true) {
            closing_from_pop.add(ref_obj)
            entry.dialog.value = false
          } else {
            // 既に閉じている残骸だった場合は待ちだけ解決する
            resolve_close_waiter(ref_obj)
          }
          count--
        }
        if (stack.length === 0) clear_dialog_keys_from_current_state()
        after_nav_settled()
        return
      }

      // E) Dialog-only mode: when stack is empty, block back navigation.
      //    This keeps the user on the current route; back is reserved for dialogs.
      if (back_only_enabled && stack.length === 0) {
        try {
          (e as PopStateEvent).stopImmediatePropagation?.()
        } catch {
          // ignore
        }

        // Cancel this back by going forward one step.
        // Mark so we don't loop.
        back_only_bounce_pending = 1
        history.go(1)
        return
      }

      after_nav_settled()
    },
    { capture: true } as AddEventListenerOptions,
  )
}

// --- Escape handling (close only topmost) ---
let esc_listener_installed = false
function ensure_esc_listener_installed() {
  if (esc_listener_installed) return
  esc_listener_installed = true

  window.addEventListener(
    "keydown",
    (e: KeyboardEvent) => {
      if (e.key !== "Escape") return
      if (e.repeat) return
      if (stack.length === 0) return

      // popstate の処理中にさらに閉じるとややこしいので避ける
      if (pending_nav > 0) return

      e.preventDefault()
      e.stopPropagation()

      // 1回の ESC で 1つだけ閉じる (×ボタンと同じく history 駆動)
      close_top_dialog()
    },
    { capture: true },
  )
}

// --- Core hook ---
export function useDialogHistoryStack(
  dialog: Ref<boolean>,
  options?: UseDialogHistoryStackOptions,
): void {
  const ref_obj = dialog as unknown as object
  const id = get_ref_id(ref_obj)

  // 同じコンポーネントの useFloatingDialog と結ぶための鍵。
  // これで「クリックで前面へ出したダイアログ」をバック/Escape の対象にできる
  const owner = getCurrentInstance() as unknown as object | null
  if (owner !== null) ref_owner_map.set(ref_obj, owner)

  if (options?.onClosed) onClosedMap.set(ref_obj, options.onClosed)

  if (watched_refs.has(ref_obj)) {
    ensure_pop_listener_installed()
    ensure_esc_listener_installed()
    return
  }
  watched_refs.add(ref_obj)
  ensure_pop_listener_installed()
  ensure_esc_listener_installed()

  const stop = watch(
    dialog,
    (open) => {
      if (open) {
        if (nav_in_flight()) {
          queue_open(id, dialog)
          return
        }
        schedule_open(id, dialog, ref_obj)
        return
      }

      // ---- close ----
      if (closing_from_pop.has(ref_obj)) {
        // close (pop): popstate ハンドラ側で stack から除去済みのことが多い
        closing_from_pop.delete(ref_obj)
        const idx = stack.findIndex((en) => en.id === id)
        if (idx >= 0) stack.splice(idx, 1)
        if (stack.length === 0) clear_dialog_keys_from_current_state()
      } else if (closing_from_reset.has(ref_obj)) {
        // close (reset)
        closing_from_reset.delete(ref_obj)
        if (stack.length === 0) clear_dialog_keys_from_current_state()
      } else if (pending_open_seq_map.has(ref_obj)) {
        // Cancel pending open if dialog closed before setTimeout fired
        pending_open_seq_map.delete(ref_obj)
        if (stack.length === 0) clear_dialog_keys_from_current_state()
      } else {
        const idx = stack.findIndex((en) => en.id === id)
        if (idx >= 0) {
          // Programmatic close (ref 直接書き換え)。該当エントリのみ除去して
          // 1エントリ巻き戻す (上に乗っているダイアログはそのまま維持される)。
          stack.splice(idx, 1)

          const target_idx = pending_close_targets.indexOf(ref_obj)
          if (target_idx >= 0) {
            // close_dialog_via_history の traversal が飛行中に直接 false にされた:
            // 二重に戻らず、着弾する popstate を握りつぶしに変換する
            pending_close_targets.splice(target_idx, 1)
            pending_close_nav = Math.max(0, pending_close_nav - 1)
            pending_nav += 1
          } else {
            pending_nav += 1
            history.go(-1)
          }
        } else if (stack.length === 0) {
          clear_dialog_keys_from_current_state()
        }
      }

      resolve_close_waiter(ref_obj)
      onClosedMap.get(ref_obj)?.()
    },
    { flush: "post" },
  )

  onBeforeUnmount(() => {
    stop()
    watched_refs.delete(ref_obj)
    closing_from_pop.delete(ref_obj)
    closing_from_reset.delete(ref_obj)
    pending_open_seq_map.delete(ref_obj)
    onClosedMap.delete(ref_obj)

    if (dialog.value === true) dialog.value = false

    const idx = stack.findIndex((en) => en.id === id)
    if (idx >= 0) stack.splice(idx, 1)

    // 飛行中の close traversal は unmount 後に着弾するため握りつぶしに変換
    const target_idx = pending_close_targets.indexOf(ref_obj)
    if (target_idx >= 0) {
      pending_close_targets.splice(target_idx, 1)
      pending_close_nav = Math.max(0, pending_close_nav - 1)
      pending_nav += 1
    }

    if (stack.length === 0) clear_dialog_keys_from_current_state()
    resolve_close_waiter(ref_obj)
  })

  onMounted(() => {
    if (dialog.value === true) {
      if (nav_in_flight()) {
        queue_open(id, dialog)
        return
      }
      schedule_open(id, dialog, ref_obj)
    }
  })
}
