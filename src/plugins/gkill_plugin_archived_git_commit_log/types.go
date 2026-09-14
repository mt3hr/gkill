package main

import "time"

const (
	// repName は manifest.json の rep_name と一致させること。
	//
	// このプラグインの記録はリポジトリごとの名前（Kyou.RepName = ディレクトリ名）を名乗り、
	// get_rep_name の rep_names でその一覧を申告する。manifest の名前は gkill が
	// 設定画面のプラグインを引き当てるための識別子で、rep の一覧には出ない。
	repName = "ArchivedGit"
	// dataType は manifest.json の data_type と一致させること。
	// native の git rep と同じ値にすることで、クライアントは native と同じ GitCommitLogView で描き、
	// 稼働中リポジトリと重なるコミットは (ID, data_type, related_time) の重複除去で1件に畳まれる。
	dataType = "git_commit_log"
	// appName はログ出力に使う。
	appName = "gkill_plugin_archived_git_commit_log"
	// gitAppName は Kyou の CreateApp / UpdateApp。native の git rep と同じ "git"。
	gitAppName = "git"
)

// config.json のキー。
const (
	configKeySourceDirs  = "source_dirs"
	configKeyMaxGitDirMB = "max_git_dir_mb"
)

// JSONにはコメントが書けないので、読み飛ばされるキーで書式を書き残す。
const (
	configKeyComment           = "_comment"
	configKeyExampleSourceDirs = "_example_source_dirs"
)

// defaultMaxGitDirMB は1リポジトリの .git の合計サイズの上限（MB）。
// 実データの最大は 6.2MB（zip 全体）で、.git だけならもっと小さい。
// 上限を超えるリポジトリはメモリに載せずに飛ばし、設定画面に理由を出す。
const defaultMaxGitDirMB = 256

// gitDirName は Git リポジトリのメタデータディレクトリ名。
const gitDirName = ".git"

// commitRecord は Git リポジトリから読んだコミット1件。
type commitRecord struct {
	Hash          string
	CommitterUnix int64
	// TZOffsetSec はコミッタ日時のタイムゾーンオフセット（秒）。
	// native の git rep は go-git が返す時刻（コミット固有のゾーン付き）をそのまま使うので、
	// ここでも UNIX 秒とオフセットの組で持って同じ instant・同じゾーンに戻す。
	TZOffsetSec int
	AuthorName  string
	AuthorEmail string
	Message     string
	Addition    int
	Deletion    int
	Files       []fileStat
	// StatsError は行数集計に失敗したときの理由。空なら成功。
	// 失敗してもコミット自体は記録する（行数だけ 0 になる）。
	StatsError string
}

// fileStat はコミットで変更したファイル1つの行数。
type fileStat struct {
	Path     string `json:"path"`
	Addition int    `json:"addition"`
	Deletion int    `json:"deletion"`
}

// commitRow はキャッシュから読んだコミット1件（一覧に必要なぶん）。
type commitRow struct {
	Hash          string
	RepName       string
	CommitterUnix int64
	TZOffsetSec   int
	AuthorName    string
	AuthorEmail   string
	Message       string
	Addition      int
	Deletion      int
}

// committedAt はコミッタ日時をコミット固有のゾーン付きで返す。
func (r commitRow) committedAt() time.Time {
	return time.Unix(r.CommitterUnix, 0).In(time.FixedZone("", r.TZOffsetSec))
}

// commitBody は詳細ビュー用の1件。一覧では読まない。
type commitBody struct {
	commitRow
	Files      []fileStat
	StatsError string
	// Sources はこのコミットが入っていた zip とその中のリポジトリの場所。
	Sources []commitSource
}

// commitSource はコミットの出どころ（zip の絶対パスと zip 内の .git の場所）。
type commitSource struct {
	ArchivePath string
	GitDir      string
}
