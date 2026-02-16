// use-dialog-history-stack.ts
import { watch, onBeforeUnmount, type Ref } from "vue"

type Entry = { id: string; dialog: Ref<boolean> }
const stack: Entry[] = []

let listening = false
let suppressNextPop = 0

const closingByPop = new Set<string>()
const closingByCascade = new Set<string>()

const MARK = "__gkillDlg"
const DEPTH = "__gkillDlgDepth"

function makeId() {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const c = (globalThis as any).crypto
  return c?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function isObj(v: unknown): v is Record<string, any> {
  return !!v && typeof v === "object"
}

function getRouterLocationString(): string {
  return `${location.pathname}${location.search}${location.hash}`
}

function getDepthFromState(state: any): number {
  if (!state || state[MARK] !== true) return 0
  const n = state?.[DEPTH]
  return typeof n === "number" && Number.isFinite(n) ? n : 0
}

function buildDialogState(depth: number) {
  // router互換っぽい形でマージ（routerのstateを壊さない）
  const base = isObj(history.state) ? history.state : {}
  const pos = typeof base.position === "number" ? base.position : 0
  const current = typeof base.current === "string" ? base.current : getRouterLocationString()

  return {
    ...base,
    back: current,
    current,
    forward: null,
    position: pos + 1,
    [MARK]: true,
    [DEPTH]: depth,
  }
}

function clearDialogKeysFromCurrentState() {
  const base = isObj(history.state) ? { ...history.state } : {}
  if (base[MARK] !== true && base[DEPTH] == null) return
  delete base[MARK]
  delete base[DEPTH]
  history.replaceState(base, "")
}

function pushDepth(depth: number) {
  history.pushState(buildDialogState(depth), "")
}

function ensureListener() {
  if (listening) return
  // capture: routerより先に拾う（重要）
  window.addEventListener("popstate", onPopState, { capture: true })
  listening = true
}

function maybeRemoveListener() {
  if (!listening) return
  if (stack.length > 0) return
  window.removeEventListener("popstate", onPopState, { capture: true } as any)
  listening = false
}

/**
 * popstate（ブラウザバック/フォワード）でダイアログを閉じる。
 * - back: stackの深さが減る → ダイアログを閉じ、routerに渡さない
 * - forward: stackの深さが増える → 事故りやすいので即座に無効化して戻す
 */
function onPopState(e: PopStateEvent) {
  if (suppressNextPop > 0) {
    suppressNextPop--
    return
  }

  const targetDepth = getDepthFromState(e.state)
  const curDepth = stack.length

  // forward（深くなるpop）は無効化：即戻す
  if (targetDepth > curDepth) {
    e.stopImmediatePropagation()
    suppressNextPop++
    history.go(-1)
    return
  }

  // back（深さが減るpop）は閉じるためなのでrouterへ渡さない
  if (curDepth > 0 && targetDepth < curDepth) {
    e.stopImmediatePropagation()
  }

  // targetDepth まで上から閉じる
  for (let i = curDepth - 1; i >= targetDepth; i--) {
    const top = stack[i]
    if (!top) continue
    closingByPop.add(top.id)
    top.dialog.value = false
  }

  stack.length = Math.min(curDepth, targetDepth)

  if (stack.length === 0) {
    // state汚染を掃除（余計なpushStateはしない：PWAの「戻るで終了」を壊しやすい）
    clearDialogKeysFromCurrentState()
  }

  maybeRemoveListener()
}

export function useDialogHistoryStack(dialog: Ref<boolean>) {
  const id = makeId()

  watch(dialog, (open) => {
    if (open) {
      if (!stack.some((e) => e.id === id)) stack.push({ id, dialog })
      pushDepth(stack.length)
      ensureListener()
      return
    }

    // pop由来のcloseは履歴操作しない（onPopState側でstackを更新済み）
    if (closingByPop.has(id)) {
      closingByPop.delete(id)
      maybeRemoveListener()
      return
    }

    // 親closeに巻き込まれた子も履歴操作しない（親側でまとめる）
    if (closingByCascade.has(id)) {
      closingByCascade.delete(id)
      maybeRemoveListener()
      return
    }

    // プログラム的に閉じた場合（例: ダイアログ外クリック）
    const idx = stack.findIndex((e) => e.id === id)
    if (idx === -1) {
      maybeRemoveListener()
      return
    }

    const prevDepth = stack.length

    // 親が閉じたなら子も閉じる（上をまとめて閉じる）
    if (idx < prevDepth - 1) {
      for (let i = prevDepth - 1; i > idx; i--) {
        const top = stack[i]
        closingByCascade.add(top.id)
        top.dialog.value = false
      }
    }

    // 自分＋子をstackから消す
    stack.splice(idx)
    const newDepth = stack.length
    const delta = prevDepth - newDepth

    // 「今いる履歴がダイアログで積んだ状態」なら、その分だけ戻す
    const histDepth = getDepthFromState(history.state)
    if (delta > 0 && histDepth === prevDepth) {
      suppressNextPop++
      history.go(-delta)
    }

    if (stack.length === 0) {
      clearDialogKeysFromCurrentState()
    }

    maybeRemoveListener()
  })

  onBeforeUnmount(() => {
    closingByPop.delete(id)
    closingByCascade.delete(id)

    const idx = stack.findIndex((e) => e.id === id)
    if (idx === -1) {
      maybeRemoveListener()
      return
    }

    const prevDepth = stack.length

    // 自分が親なら子も一緒に閉じる
    if (idx < prevDepth - 1) {
      for (let i = prevDepth - 1; i > idx; i--) {
        const top = stack[i]
        closingByCascade.add(top.id)
        top.dialog.value = false
      }
    }

    stack.splice(idx)
    const newDepth = stack.length
    const delta = prevDepth - newDepth

    const histDepth = getDepthFromState(history.state)
    if (delta > 0 && histDepth === prevDepth) {
      suppressNextPop++
      history.go(-delta)
    }

    if (stack.length === 0) {
      clearDialogKeysFromCurrentState()
    }

    maybeRemoveListener()
  })
}
