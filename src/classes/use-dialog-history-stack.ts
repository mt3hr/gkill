import { onBeforeUnmount, onMounted, type Ref, watch } from "vue"

/**
 * Dialog history stack manager with race protection (optimized).
 *
 * Key optimizations vs previous version:
 * - history.replaceState is wrapped ONLY while dialogs are open; restored when stack becomes empty.
 * - The wrapper avoids cloning state unless necessary.
 * - watch() uses default flush (no "sync") to reduce synchronous overhead.
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
  // Avoid cloning if keys are not present
  if (!(MARK in state) && !(DEPTH in state)) return state
  const { [MARK]: _m, [DEPTH]: _d, ...rest } = state
  return rest
}

/**
 * Try to "apply" dialog markers to an existing state object without cloning.
 * If we cannot safely mutate, we clone minimally.
 */
function applyDialogMarkers(base: any, depth: number): any {
  if (!isObj(base)) {
    return { [MARK]: true, [DEPTH]: depth }
  }

  // If already correct, reuse object
  if (base[MARK] === true && base[DEPTH] === depth) return base

  // Prefer in-place mutation if possible
  try {
    // Some objects might be non-extensible/frozen; assignment will throw in strict mode.
    (base as AnyObj)[MARK] = true
      ; (base as AnyObj)[DEPTH] = depth
    return base
  } catch {
    // Fallback: shallow clone
    const st = { ...(base as AnyObj) }
    st[MARK] = true
    st[DEPTH] = depth
    return st
  }
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

// --- Race protection: queue opens while a history.go(-delta) close is pending ---
let pendingNav = 0
const queuedOpens: Array<{ id: string; dialog: Ref<boolean> }> = []

function queueOpen(id: string, dialog: Ref<boolean>) {
  const idx = queuedOpens.findIndex((x) => x.id === id)
  if (idx >= 0) queuedOpens.splice(idx, 1)
  queuedOpens.push({ id, dialog })
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
    history.pushState(applyDialogMarkers(history.state, stack.length), "")
  }
}

// --- replaceState guard (installed only while dialogs are open) ---
let rawReplaceState: History["replaceState"] | null = null
let guardActive = false

function activateReplaceGuard() {
  if (guardActive) return
  guardActive = true

  if (!rawReplaceState) rawReplaceState = history.replaceState.bind(history)

  history.replaceState = ((data: any, unused: string, url?: string | URL | null) => {
    // While dialogs are open, preserve markers with current stack length.
    // Do NOT clone unless needed.
    const next = applyDialogMarkers(data, stack.length)
    return rawReplaceState!(next, unused, url as any)
  }) as any
}

function deactivateReplaceGuardIfPossible() {
  if (!guardActive) return
  if (stack.length !== 0) return
  // Restore native replaceState
  if (rawReplaceState) {
    history.replaceState = rawReplaceState as any
  }
  guardActive = false
}

function clearDialogKeysFromCurrentState() {
  if (stack.length !== 0) return
  // Ensure we are NOT guarding replaceState here to avoid overhead and marker re-adding.
  deactivateReplaceGuardIfPossible()
  const cleaned = stripDialogKeys(history.state)
  // Use rawReplaceState if available, otherwise current replaceState
  const rs = rawReplaceState ? rawReplaceState : history.replaceState.bind(history)
  rs(cleaned, "")
}

// --- popstate handling ---
let popListenerInstalled = false
function ensurePopListenerInstalled() {
  if (popListenerInstalled) return
  popListenerInstalled = true

  window.addEventListener("popstate", (e) => {
    if (pendingNav > 0) {
      pendingNav = 0
      if (stack.length === 0) clearDialogKeysFromCurrentState()
      flushQueuedOpens()
      return
    }

    // Forward into a dialog state while stack is empty: block it.
    if (stack.length === 0 && isDialogState(e.state)) {
      pendingNav = 1
      history.go(-1)
      return
    }

    // If any dialog is open, back/forward closes the topmost.
    if (stack.length > 0) {
      const top = stack[stack.length - 1]
      closingFromPop.add(top.dialog as any)
      top.dialog.value = false
      return
    }
  })
}

// --- Core hook ---
export function useDialogHistoryStack(dialog: Ref<boolean>): void {
  const refObj = dialog as unknown as object
  const id = getRefId(refObj)

  if (watchedRefs.has(refObj)) {
    // Still ensure pop listener
    ensurePopListenerInstalled()
    return
  }
  watchedRefs.add(refObj)
  ensurePopListenerInstalled()

  const stop = watch(dialog, (open) => {
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

      // Turn on replaceState guard only while dialogs are open
      activateReplaceGuard()

      history.pushState(applyDialogMarkers(history.state, stack.length), "")
      return
    }

    // close
    if (closingFromPop.has(refObj)) {
      closingFromPop.delete(refObj)
      const idx = stack.findIndex((e) => e.id === id)
      if (idx >= 0) stack.splice(idx, 1)
      if (stack.length === 0) clearDialogKeysFromCurrentState()
      return
    }

    // Programmatic close: remove itself and children, rewind history.
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
    }

    if (stack.length === 0 && delta === 0) {
      clearDialogKeysFromCurrentState()
    }
  })

  onBeforeUnmount(() => {
    stop()
    watchedRefs.delete(refObj)

    // If unmounted while open, close it.
    if (dialog.value === true) dialog.value = false

    const idx = stack.findIndex((e) => e.id === id)
    if (idx >= 0) stack.splice(idx, 1)
    if (stack.length === 0) clearDialogKeysFromCurrentState()
  })

  onMounted(() => {
    if (dialog.value === true) {
      if (pendingNav > 0) {
        queueOpen(id, dialog)
      } else {
        const existingIdx = stack.findIndex((e) => e.id === id)
        if (existingIdx >= 0) {
          const [e] = stack.splice(existingIdx, 1)
          stack.push(e)
        } else {
          stack.push({ id, dialog })
        }
        activateReplaceGuard()
        history.pushState(applyDialogMarkers(history.state, stack.length), "")
      }
    }
  })
}
