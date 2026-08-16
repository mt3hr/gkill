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

/**
 * ツリーの「入れ物」ノード（ルートとフォルダ）かどうか。
 *
 * 入れ物は並べ替えのための器でしかないのに、`key` にはフォルダ名が
 * （ルートは `__root__` が）そのまま入る。チェックの入ったノードの key は
 * **そのまま検索条件（タグ名 / リポジトリ名 / 端末名 / 記録種別名）として流れる**ので、
 * 入れ物を混ぜると「そんな名前のタグは存在しない」という条件が紛れ込む。
 * OR検索では無害だが、**AND検索（`tags_and` 等）では必ず0件になる**。
 * ルート行は folder_name='' の空白帯として描かれていてクリックできてしまうため、
 * `__root__` は誤クリックだけで条件に入る。
 *
 * 実データを持つのは必ず葉。フォルダは編集ダイアログの「フォルダ追加」でしか作られず
 * （`add_folder_struct_element` が `is_dir=true` 固定で新規ノードを足す）、
 * 葉が後から入れ物に変わることはない。フォルダ名と同名のタグが実在する場合も、
 * `apply_check_state_to_struct` が key 一致でツリー全体を走査して葉のほうにも
 * チェックを入れるので、入れ物を除いても条件は落ちない。
 */
export function is_struct_container_node(struct: FoldableStructModel): boolean {
    return struct.is_dir || struct.key === FOLDABLE_STRUCT_ROOT_KEY
}