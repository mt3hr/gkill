import { onBeforeUnmount, onMounted, type Ref, watch } from "vue"

/**
 * Dialog history stack manager (lightweight, router-friendly)
 * + resetDialogHistory()
 * + Back is dialog-only mode (when stack is empty, back is blocked).
 *
 * Notes:
 * - This will prevent leaving the app via back button (PWA exit / previous site).
 * - Forward into dialog states while stack is empty is also blocked.
 */

const MARK = "__gkillDlg"
const DEPTH = "__gkillDlgDepth"

type AnyObj = Record<string, any>
function isObj(v: any): v is AnyObj {
  return v !== null && typeof v === "object"
}
function isDialogState(state: any): boolean {
  return isObj(state) && state[MARK] === true && typeof state[DEPTH] === "number"
}
function stripDialogKeys(state: any): any {
  if (!isObj(state)) return state
  if (!(MARK in state) && !(DEPTH in state)) return state
  const { [MARK]: _m, [DEPTH]: _d, ...rest } = state
  return rest
}
function withDialogMarkers(base: any, depth: number): any {
  const b: AnyObj = isObj(base) ? (base as AnyObj) : {}
  return { ...b, [MARK]: true, [DEPTH]: depth }
}

// --- Global stack (module singleton) ---
type Entry = { id: string; dialog: Ref<boolean> }
const stack: Entry[] = []

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

// --- Race protection: queue opens while a history.go(-delta) close is pending ---
let pendingNav = 0
const queuedOpens: Array<{ id: string; dialog: Ref<boolean> }> = []

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
  if (pendingNav > 0 || queuedOpens.length === 0) return
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

/**
 * Close all dialogs and rewind browser history by the dialog depth.
 */
export function resetDialogHistory(): Promise<void> {
  if (pendingNav === 0 && stack.length === 0) return Promise.resolve()

  const depth = stack.length
  return new Promise<void>((resolve) => {
    resetWaiters.push(resolve)

    if (depth <= 0) {
      if (pendingNav === 0) resolveResetWaiters()
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

    pendingNav = depth
    history.go(-depth)
  })
}

// --- Back is dialog-only mode ---
const backOnlyEnabled = true
let backOnlyBouncePending = 0 // prevents infinite loops when we call history.go(1)

// --- popstate handling ---
let popListenerInstalled = false
function ensurePopListenerInstalled() {
  if (popListenerInstalled) return
  popListenerInstalled = true

  window.addEventListener(
    "popstate",
    (e) => {
      // A) If this popstate was caused by our own history.go(+/-), swallow it.
      if (pendingNav > 0) {
        pendingNav = 0
        if (stack.length === 0) clearDialogKeysFromCurrentState()
        flushQueuedOpens()
        resolveResetWaiters()
        return
      }

      // B) Back-only bounce (we called history.go(1) to cancel a back)
      if (backOnlyBouncePending > 0) {
        backOnlyBouncePending = 0
        return
      }

      // C) Block forward into dialog states while stack is empty
      if (stack.length === 0 && isDialogState((e as PopStateEvent).state)) {
        pendingNav = 1
        history.go(-1)
        return
      }

      // D) If any dialog is open, back/forward closes the topmost.
      if (stack.length > 0) {
        try {
          (e as any).stopImmediatePropagation?.()
        } catch {
          // ignore
        }

        const top = stack[stack.length - 1]
        closingFromPop.add(top.dialog as any)
        top.dialog.value = false
        return
      }

      // E) Dialog-only mode: when stack is empty, block back navigation.
      //    This keeps the user on the current route; back is reserved for dialogs.
      if (backOnlyEnabled && stack.length === 0) {
        try {
          (e as any).stopImmediatePropagation?.()
        } catch {
          // ignore
        }

        // Cancel this back by going forward one step.
        // Mark so we don't loop.
        backOnlyBouncePending = 1
        history.go(1)
        return
      }
    },
    { capture: true } as any,
  )
}

// --- Core hook ---
export function useDialogHistoryStack(dialog: Ref<boolean>): void {
  const refObj = dialog as unknown as object
  const id = getRefId(refObj)

  if (watchedRefs.has(refObj)) {
    ensurePopListenerInstalled()
    return
  }
  watchedRefs.add(refObj)
  ensurePopListenerInstalled()

  const stop = watch(
    dialog,
    (open) => {
      if (open) {
        if (pendingNav > 0) {
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
        return
      }

      // close (pop)
      if (closingFromPop.has(refObj)) {
        closingFromPop.delete(refObj)
        const idx = stack.findIndex((e) => e.id === id)
        if (idx >= 0) stack.splice(idx, 1)
        if (stack.length === 0) clearDialogKeysFromCurrentState()
        return
      }

      // close (reset)
      if (closingFromReset.has(refObj)) {
        closingFromReset.delete(refObj)
        if (stack.length === 0) clearDialogKeysFromCurrentState()
        return
      }

      // Programmatic close
      const idx = stack.findIndex((e) => e.id === id)
      if (idx < 0) {
        if (stack.length === 0) clearDialogKeysFromCurrentState()
        return
      }

      const prevDepth = stack.length
      stack.splice(idx, stack.length - idx)
      const delta = prevDepth - stack.length

      if (delta > 0) {
        pendingNav = delta
        history.go(-delta)
      } else if (stack.length === 0) {
        clearDialogKeysFromCurrentState()
      }
    },
    { flush: "post" },
  )

  onBeforeUnmount(() => {
    stop()
    watchedRefs.delete(refObj)
    closingFromPop.delete(refObj)
    closingFromReset.delete(refObj)

    if (dialog.value === true) dialog.value = false

    const idx = stack.findIndex((e) => e.id === id)
    if (idx >= 0) stack.splice(idx, 1)
    if (stack.length === 0) clearDialogKeysFromCurrentState()
  })

  onMounted(() => {
    if (dialog.value === true) {
      if (pendingNav > 0) {
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
    }
  })
}
