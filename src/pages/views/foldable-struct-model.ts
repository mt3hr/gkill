export interface FoldableStructModel {
    name: string
    id: string | null
    children: Array<FoldableStructModel> | null
    key: string
    is_checked: boolean
    indeterminate: boolean
    is_dir: boolean
}