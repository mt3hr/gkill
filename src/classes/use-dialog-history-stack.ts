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

// 「空になったら forward を切り捨てる」予約（軽量：空のときだけ1回pushState）
let truncateWhenEmpty = false
let truncateScheduled = false

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
    const base = isObj(history.state) ? history.state : {}
    const pos = typeof base.position === "number" ? base.position : 0
    const current = typeof base.current === "string" ? base.current : getRouterLocationString()

    // router互換っぽい形でマージ（routerのstateを壊さない）
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

function buildBaseStateToTruncateForward() {
    // forward切り捨て用：MARK/DEPTH を外したベース状態を push して future を捨てる
    const base = isObj(history.state) ? history.state : {}
    const pos = typeof base.position === "number" ? base.position : 0
    const current = typeof base.current === "string" ? base.current : getRouterLocationString()

    const next: any = { ...base }
    delete next[MARK]
    delete next[DEPTH]

    return {
        ...next,
        back: current,
        current,
        forward: null,
        position: pos + 1,
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
    // capture: routerより先に拾う（超重要）
    window.addEventListener("popstate", onPopState, { capture: true })
    listening = true
}

function maybeRemoveListener() {
    if (!listening) return
    if (stack.length > 0) return
    window.removeEventListener("popstate", onPopState, { capture: true } as any)
    listening = false
}

function scheduleTruncateForwardWhenEmpty() {
    if (!truncateWhenEmpty) return
    if (truncateScheduled) return
    truncateScheduled = true

    // router.replace と同tickでやるとUI(カーソル等)が揺れやすいので次tick
    setTimeout(() => {
        truncateScheduled = false
        if (!truncateWhenEmpty) return
        if (stack.length !== 0) return

        truncateWhenEmpty = false
        clearDialogKeysFromCurrentState()
        history.pushState(buildBaseStateToTruncateForward(), "")
    }, 0)
}

function onPopState(e: PopStateEvent) {
    if (suppressNextPop > 0) {
        suppressNextPop--
        return
    }

    const targetDepth = getDepthFromState(e.state)
    const curDepth = stack.length

    // forward（深くなるpop）は「無効化」する：戻して終了（軽い）
    if (targetDepth > curDepth) {
        e.stopImmediatePropagation()
        suppressNextPop++
        history.go(-1)
        // 空の時に1回だけfutureを切り捨てたいなら予約
        if (curDepth === 0) {
            truncateWhenEmpty = true
            scheduleTruncateForwardWhenEmpty()
        }
        return
    }

    // back（深さが減るpop）は「閉じるため」なので router に渡さない
    if (curDepth > 0 && targetDepth < curDepth) {
        e.stopImmediatePropagation()
    }

    // targetDepth まで上から閉じる
    for (let i = stack.length - 1; i >= targetDepth; i--) {
        const top = stack[i]
        if (!top) continue
        closingByPop.add(top.id)
        top.dialog.value = false
    }

    stack.length = Math.min(stack.length, targetDepth)

    if (stack.length === 0) {
        // state汚染を掃除して、必要ならfutureを1回だけ切り捨て
        clearDialogKeysFromCurrentState()
        truncateWhenEmpty = true
        scheduleTruncateForwardWhenEmpty()
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

        // pop由来のcloseは履歴操作しない
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

        // プログラム的に閉じた場合
        const idx = stack.findIndex((e) => e.id === id)
        if (idx === -1) {
            maybeRemoveListener()
            return
        }

        // 親が閉じたなら子も閉じる（上をまとめて閉じる）
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

        // 「自分たちの履歴の上」にいる時だけ戻す（安全策）
        const histDepth = getDepthFromState(history.state)
        if (delta > 0 && histDepth === prevDepth) {
            suppressNextPop++
            history.go(-delta)
        }

        if (stack.length === 0) {
            clearDialogKeysFromCurrentState()
            truncateWhenEmpty = true
            scheduleTruncateForwardWhenEmpty()
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
            truncateWhenEmpty = true
            scheduleTruncateForwardWhenEmpty()
        }

        maybeRemoveListener()
    })
}
