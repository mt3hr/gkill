export interface FoldableStructModel {
    name: string
    seq_in_parent: number
    id: string | null
    children: Array<FoldableStructModel> | null
    key: string
    is_checked: boolean
    indeterminate: boolean
}