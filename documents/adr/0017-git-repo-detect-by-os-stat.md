# ADR-0017: gitリポジトリ判定は PlainOpen のエラー型ではなく os.Stat で行う

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-20 |
| Sources | `17797717` / `081a6e44` / `77182397` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/git_commit_log_repository_local_dir_impl.go` |

## Context

`$HOME/Git/*` のような glob は zglob が**ファイルも返す**ので、Git フォルダに `bash.exe.stackdump` のような非gitのファイルが1つ混ざることがある（Git Bash のクラッシュダンプ。このリポジトリにも実在する）。

`gkill_dao_manager.go` は `errors.Is(err, reps.ErrNotGitRepository)` のときだけそのrepをスキップし、それ以外は `GetRepositories` を丸ごと失敗させる。失敗すると**そのユーザの全APIが `ERR000018` になって何もできなくなる**。

ところが `NewGitRep` は `git.PlainOpen` のエラーを `%w` で包み直すだけで、**その型が OS で違う**。

```
Windows … ErrRepositoryNotExists（判別できる）
Linux   … lstat <path>/.git: not a directory（ENOTDIR の *fs.PathError）
```

つまり Linux では `errors.Is` が偽になり、スキップされずに `return nil, err` へ落ちる。**`081a6e44` で直したはずの不具合が Linux では直っていなかった。** CI（Ubuntu）はこの2件で **5日間赤のまま**で、Windows では再現しないので手元では気付けなかった。

## Decision

gitリポジトリは必ずディレクトリなので、`PlainOpen` の**前に `os.Stat` でディレクトリかを確かめ**、ファイルなら `ErrNotGitRepository` を包んで返す。OSに依存しない判定にする。

## Rejected alternatives

- **`PlainOpen` のエラーを一律 `ErrNotGitRepository` に丸める** — 権限エラーや壊れたリポジトリまでスキップに化けて、**gitのコミットログが1件も出ないことに気付けなくなる**。`TestNewGitRepSuccessForGitRepository` がその意図で置いてある。

- **OSごとにエラー型を判定する分岐を書く** — go-git や標準ライブラリの実装詳細に依存する。次のバージョンで文言や型が変われば同じ壊れ方が再発し、しかも**そのOSでしか再現しない**ので気付くまでに日数がかかる。

- **glob 側でファイルを除外する** — zglob の挙動に依存するうえ、rep の種類ごとに同じ除外を書くことになる。判定はrepを開く側に1つあればよい。

## Consequences

**「OSによってエラーの型が違う」は静かに壊れる。** ビルドも vet も通り、片方のOSでは正しく動く。Windows で開発して Linux 向けにも配布する gkill では、この形の分岐を新しく書かないこと。

配布物のうち linux_amd64 / linux_arm64 / linux_arm / android_arm / android_arm64 のサーバがこの不具合に当たっていた。

## Evidence

- CI（Ubuntu）が `081a6e44`（2026-08-15）から **5日間赤のまま**。最後に緑だったのは `50941029`
- Windows では再現しない。WSL で修正前の失敗を再現し、修正後に Windows（30パッケージ）と Linux の両方で通ることを確認した

## Related tests

- `src/server/gkill/dao/gkill_dao_manager_git_rep_test.go`（`TestNewGitRepReturnsErrNotGitRepositoryForNonGitPath` / `TestGetRepositoriesSkipsNonGitEntriesInGitCommitLogGlob`）
