package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	gitcache "github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// zip の中の .git は go-git で直接は開けない。
// packfile の読み取りに ReadAt / Seek が要り、archive/zip のエントリはストリームでしか読めないため。
// そこで .git 配下のエントリを go-billy の memfs へ流し込み、それを go-git のストレージにする。
// ディスクには何も書かない。実データの .git は大きくても数MBなので、メモリで足りる。
//
// 一時ディレクトリへ展開して git.PlainOpen する案は、派生キャッシュが zip と二重にディスクを
// 食う（本体の zip_cache と同じ問題）ので採らない。

// errGitDirTooLarge は .git の合計サイズが上限を超えたことを表す。
var errGitDirTooLarge = errors.New("git directory is too large")

// readRepoCommits は1リポジトリぶんの .git エントリを読み、全 ref から辿れる全コミットを返す。
// 並びは保証しない。コミットが1件も無いリポジトリ（git init 直後）は空を返し、エラーにしない。
func readRepoCommits(ctx context.Context, bundle repoBundle, maxBytes int64) ([]commitRecord, error) {
	if bundle.TotalSize > maxBytes {
		return nil, fmt.Errorf("%w: %d bytes > %d bytes", errGitDirTooLarge, bundle.TotalSize, maxBytes)
	}

	fs := memfs.New()
	for _, entry := range bundle.Entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rel, ok := gitRelativePath(entry.EntryName)
		if !ok || rel == "" {
			continue
		}
		if err := copyEntryToFS(fs, rel, entry); err != nil {
			return nil, err
		}
	}

	storer := filesystem.NewStorage(fs, gitcache.NewObjectLRUDefault())
	repo, err := git.Open(storer, nil)
	if err != nil {
		return nil, fmt.Errorf("error at open git repository %s!/%s: %w", bundle.ArchivePath, bundle.GitDir, err)
	}

	// native の git rep と同じく全 ref（refs/remotes/* も含む）から辿る。
	iter, err := repo.Log(&git.LogOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("error at git log %s!/%s: %w", bundle.ArchivePath, bundle.GitDir, err)
	}
	defer iter.Close()

	records := []commitRecord{}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		commit, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error at iterate commits %s!/%s: %w", bundle.ArchivePath, bundle.GitDir, err)
		}
		records = append(records, recordOf(ctx, commit))
	}
	return records, nil
}

// copyEntryToFS は zip のエントリを memfs の rel に書く。親ディレクトリは memfs が作る。
func copyEntryToFS(fs billy.Filesystem, rel string, entry sdk.SourceEntry) error {
	reader, err := entry.Open()
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	file, err := fs.Create(rel)
	if err != nil {
		return fmt.Errorf("error at create %s in memory fs: %w", rel, err)
	}
	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		return fmt.Errorf("error at read zip entry %s: %w", entry.Path, err)
	}
	return file.Close()
}

// recordOf は go-git のコミットを commitRecord にする。
//
// 列の取り方は native の git rep（git_commit_log_repository_local_dir_impl.go）と同じ:
// 時刻はコミッタ日時、利用者名は author 名、メッセージは trim しない、
// 行数は StatsContext の合計（root commit は空ツリーとの差、merge は親1との差）。
// native は行数集計の失敗で検索全体を落とすが、ここは常駐ビルダなので
// そのコミットだけ行数 0 で記録し、理由を残す。
func recordOf(ctx context.Context, commit *object.Commit) commitRecord {
	_, offset := commit.Committer.When.Zone()
	record := commitRecord{
		Hash:          commit.Hash.String(),
		CommitterUnix: commit.Committer.When.Unix(),
		TZOffsetSec:   offset,
		AuthorName:    commit.Author.Name,
		AuthorEmail:   commit.Author.Email,
		Message:       commit.Message,
	}

	stats, err := commit.StatsContext(ctx)
	if err != nil {
		record.StatsError = err.Error()
		return record
	}
	record.Files = make([]fileStat, 0, len(stats))
	for _, stat := range stats {
		record.Addition += stat.Addition
		record.Deletion += stat.Deletion
		record.Files = append(record.Files, fileStat{Path: stat.Name, Addition: stat.Addition, Deletion: stat.Deletion})
	}
	slices.SortFunc(record.Files, func(a, b fileStat) int { return strings.Compare(a.Path, b.Path) })
	return record
}
