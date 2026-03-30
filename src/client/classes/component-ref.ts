/**
 * Type for Vue template refs to child components.
 *
 * Uses an index signature so that methods / properties exposed by the
 * component can be accessed via optional chaining (e.g.
 * `ref.value?.show(e)`) without resorting to bare `any` on the ref
 * declaration itself.
 *
 * The single `any` here is intentional — it is the narrowest practical
 * escape hatch that keeps every call-site type-safe *enough* while
 * avoiding circular component imports.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type ComponentRef = Record<string, any>
