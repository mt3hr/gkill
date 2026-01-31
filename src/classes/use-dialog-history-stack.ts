// use-dialog-history-stack.ts (improved)
import { watch, onBeforeUnmount, type Ref } from "vue"

type Entry = { id: string; dialog: Ref<boolean> }
const stack: Entry[] = []

let listening = false
let suppressNextPop = 0

const closingByPop = new Set<string>()
const closingByCascade = new Set<string>()

const KEY = "__gkillDlgDepth"
const MARK = "__gkillDlg" // これが true の state だけを「ダイアログ履歴」として扱う

function makeId() {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const c = (globalThis as any).crypto
    return c?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function isObj(v: unknown): v is Record<string, any> {
    return !!v && typeof v === "object"
}

function getDepthFromState(state: any): number {
    if (!state || state[MARK] !== true) return 0
    const n = state?.[KEY]
    return typeof n === "number" && Number.isFinite(n) ? n : 0
}

function getRouterLocationString(): string {
    // vue-router が state.current に入れているのは「originなし」の URL 文字列であることが多い
    return `${location.pathname}${location.search}${location.hash}`
}

function buildDialogState(depth: number) {
    const base = isObj(history.state) ? history.state : {}
    const pos = typeof base.position === "number" ? base.position : 0
    const current = typeof base.current === "string" ? base.current : getRouterLocationString()

    // 「同一URLへ push された」状態を router にそれっぽく見せる
    // back は直前 entry の current（= current）にしておくのが一番安全
    return {
        ...base,
        back: current,
        current,
        forward: null,
        position: pos + 1,
        [MARK]: true,
        [KEY]: depth,
    }
}

function pushDepth(depth: number) {
    history.pushState(buildDialogState(depth), "")
}

function clearDialogKeysFromCurrentState() {
    const base = isObj(history.state) ? { ...history.state } : {}
    if (base[MARK] !== true && base[KEY] == null) return

    delete base[MARK]
    delete base[KEY]
    history.replaceState(base, "")
}

function ensureListener() {
    if (listening) return
    // capture にして router より先に拾う
    window.addEventListener("popstate", onPopState, { capture: true })
    listening = true
}

function maybeRemoveListener() {
    if (!listening) return
    if (stack.length > 0) return
    window.removeEventListener("popstate", onPopState, { capture: true } as any)
    listening = false
}

function onPopState(e: PopStateEvent) {
    if (suppressNextPop > 0) {
        suppressNextPop--
        return
    }

    const targetDepth = getDepthFromState(e.state)
    const curDepth = stack.length

    // ダイアログが開いていて、深さが減る pop は「ダイアログを閉じるための戻る」なので router に渡さない
    if (curDepth > 0 && targetDepth < curDepth) {
        e.stopImmediatePropagation()
    }

    // 深さが減った分だけ「上から」閉じる
    for (let i = stack.length - 1; i >= targetDepth; i--) {
        const top = stack[i]
        if (!top) continue
        closingByPop.add(top.id)
        top.dialog.value = false
    }

    stack.length = Math.min(stack.length, targetDepth)

    if (stack.length === 0) {
        // 次のルート遷移に KEY が混ざらないように掃除
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

        // close
        if (closingByPop.has(id)) {
            closingByPop.delete(id)
            if (stack.length === 0) clearDialogKeysFromCurrentState()
            maybeRemoveListener()
            return
        }

        if (closingByCascade.has(id)) {
            closingByCascade.delete(id)
            if (stack.length === 0) clearDialogKeysFromCurrentState()
            maybeRemoveListener()
            return
        }

        const idx = stack.findIndex((e) => e.id === id)
        if (idx === -1) {
            if (stack.length === 0) clearDialogKeysFromCurrentState()
            maybeRemoveListener()
            return
        }

        // 親が閉じたなら子も閉じる
        if (idx < stack.length - 1) {
            for (let i = stack.length - 1; i > idx; i--) {
                const top = stack[i]
                closingByCascade.add(top.id)
                top.dialog.value = false
            }
        }

        const prevDepth = stack.length
        stack.length = idx
        const newDepth = stack.length

        const delta = prevDepth - newDepth
        const histDepth = getDepthFromState(history.state)

        // 「いま自分たちが積んだダイアログ履歴の上にいる」場合だけ戻す
        // (>= ではなく === に寄せて事故を減らす)
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
        if (idx !== -1) stack.splice(idx, 1)

        if (stack.length === 0) {
            clearDialogKeysFromCurrentState()
        }

        maybeRemoveListener()
    })
}
