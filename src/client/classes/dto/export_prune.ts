export function prune_empty(v: unknown): unknown {
    if (v === null || v === undefined) return undefined

    if (Array.isArray(v)) {
        const a = v.map(prune_empty).filter(x => x !== undefined)
        return a.length === 0 ? undefined : a
    }

    if (typeof v === "object") {
        const o: Record<string, unknown> = {}
        for (const [k, val] of Object.entries(v)) {
            const pv = prune_empty(val)
            if (pv !== undefined) o[k] = pv
        }
        return Object.keys(o).length === 0 ? undefined : o
    }

    // "" / 0 / false は残す
    return v
}
