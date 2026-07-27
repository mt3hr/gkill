// ツリーのルートノードのname/keyに入れる値。
// ルートは表示名をfolder_name propから受け取るためこの値が画面に出ることはないが、
// keyはチェック時に検索条件（タグ名やリポジトリ名）として流れるため、
// 実在するタグ名等と衝突しない値にしてある。
export const FOLDABLE_STRUCT_ROOT_KEY = "__root__"

export interface FoldableStructModel {
    name: string
    id: string | null
    children: Array<FoldableStructModel> | null
    key: string
    is_checked: boolean
    indeterminate: boolean
    is_dir: boolean
}