export function pruneEmpty(v: any): any {
    if (v === null || v === undefined) return undefined

    if (Array.isArray(v)) {
        const a = v.map(pruneEmpty).filter(x => x !== undefined)
        return a.length === 0 ? undefined : a
    }

    if (typeof v === "object") {
        const o: any = {}
        for (const [k, val] of Object.entries(v)) {
            const pv = pruneEmpty(val)
            if (pv !== undefined) o[k] = pv
        }
        return Object.keys(o).length === 0 ? undefined : o
    }

    // "" / 0 / false は残す
    return v
}