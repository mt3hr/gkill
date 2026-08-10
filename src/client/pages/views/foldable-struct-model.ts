// ツリーのルートノードのname/keyに入れる値。
// ルートは表示名をfolder_name propから受け取るためこの値が画面に出ることはないが、
// keyはチェック時に検索条件（タグ名やリポジトリ名）として流れるため、
// 実在するタグ名等と衝突しない値にしてある。
export const FOLDABLE_STRUCT_ROOT_KEY = "__root__"

/**
 * FoldableStruct が扱うツリーの共通形。
 *
 * **並び順は `children` 配列の順そのもの**、**折りたたみはルートだけ開いた状態から始まる**。
 * これが全ツリー（板 / タグ / rep / 端末 / rep種別 / KFTLテンプレート）共通の仕様で、
 * `use-foldable-struct.ts` の `updated_struct()` と `open_group` がその実体。
 *
 * 以前は `seq` / `seq_in_parent` / `is_open_default` という並び順・折りたたみ用のフィールドが
 * 各 ElementData に散らばっていたが、**どれも書かれるだけで一度も読まれていなかった**
 * （しかも持っているフィールドがツリーごとにバラバラだった）ので削除した。
 * 保存済みのJSONにはまだ残っているが、読み捨てられる。
 * 並べ替えは `classes/foldable-struct-move.ts` が `children` を直接組み替えて行う。
 */
export interface FoldableStructModel {
    name: string
    id: string | null
    children: Array<FoldableStructModel> | null
    key: string
    is_checked: boolean
    indeterminate: boolean
    is_dir: boolean
}