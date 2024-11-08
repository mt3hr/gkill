export interface FoldableStructModel {
    id: string
    children: Array<FoldableStructModel> | null
    key: string
    is_checked: boolean
    indeterminate: boolean
}