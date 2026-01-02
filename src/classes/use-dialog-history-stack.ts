// useDialogBackStack.ts
import { watch, onBeforeUnmount, type Ref } from "vue"

type Entry = { id: string; dialog: Ref<boolean> }
const stack: Entry[] = []

let listening = false
let suppressNextPop = 0 // 「掃除でhistory.go/backした popstate」を無視する
const closingByPop = new Set<string>() // popstate由来のcloseか判定
const closingByCascade = new Set<string>() // 親closeに巻き込まれたcloseか判定

const KEY = "__gkillDlgDepth"

function makeId() {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const c = (globalThis as any).crypto
    return c?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function getDepthFromState(state: any): number {
    const n = state?.[KEY]
    return typeof n === "number" && Number.isFinite(n) ? n : 0
}

function pushDepth(depth: number) {
    // ★重要：routerが使っているhistory.stateを壊さないように必ずマージする
    const base =
        history.state && typeof history.state === "object" ? history.state : {}
    history.pushState({ ...base, [KEY]: depth }, "")
}

function ensureListener() {
    if (listening) return
    window.addEventListener("popstate", onPopState, { passive: true })
    listening = true
}

function maybeRemoveListener() {
    if (!listening) return
    if (stack.length > 0) return
    window.removeEventListener("popstate", onPopState)
    listening = false
}

function onPopState(e: PopStateEvent) {
    if (suppressNextPop > 0) {
        suppressNextPop--
        return
    }

    const targetDepth = getDepthFromState(e.state)

    // 深さが減った分だけ「上から」閉じる（入れ子対応）
    for (let i = stack.length - 1; i >= targetDepth; i--) {
        const top = stack[i]
        if (!top) continue
        closingByPop.add(top.id)
        top.dialog.value = false
    }

    // stack自体も合わせる（watch側で見つからなくてもOKな作りにしてる）
    stack.length = Math.min(stack.length, targetDepth)
    maybeRemoveListener()
}

export function useDialogHistoryStack(dialog: Ref<boolean>) {
    const id = makeId()

    watch(dialog, (open) => {
        if (open) {
            // 既に登録済みなら二重登録しない
            if (!stack.some((e) => e.id === id)) stack.push({ id, dialog })

            // 開くたびに「戻る1段」を差し込む（URLは変えない）
            pushDepth(stack.length)

            ensureListener()
            return
        }

        // close
        const fromPop = closingByPop.has(id)
        if (fromPop) {
            closingByPop.delete(id)
            // popstate由来のcloseでは履歴操作しない
            maybeRemoveListener()
            return
        }

        const fromCascade = closingByCascade.has(id)
        if (fromCascade) {
            closingByCascade.delete(id)
            // 親closeに巻き込まれた子は、履歴操作は親側でまとめてやる
            maybeRemoveListener()
            return
        }

        const idx = stack.findIndex((e) => e.id === id)
        if (idx === -1) {
            maybeRemoveListener()
            return
        }

        // もし「下の階層（親）」が閉じられたなら、上の階層（子）もまとめて閉じる
        if (idx < stack.length - 1) {
            for (let i = stack.length - 1; i > idx; i--) {
                const top = stack[i]
                closingByCascade.add(top.id)
                top.dialog.value = false
            }
        }

        // スタックから自分と上を削る
        const prevDepth = stack.length
        stack.length = idx // idxより上は閉じた扱い
        const newDepth = stack.length

        // 履歴もまとめて戻す（popstate連鎖を無視する）
        // ※history.stateが自分たちの深さを持っている時だけ戻す（安全策）
        const curDepth = getDepthFromState(history.state)
        const delta = prevDepth - newDepth
        if (delta > 0 && curDepth >= prevDepth) {
            suppressNextPop++
            history.go(-delta)
        }

        maybeRemoveListener()
    })

    onBeforeUnmount(() => {
        closingByPop.delete(id)
        closingByCascade.delete(id)
        const idx = stack.findIndex((e) => e.id === id)
        if (idx !== -1) stack.splice(idx, 1)
        maybeRemoveListener()
    })
}
