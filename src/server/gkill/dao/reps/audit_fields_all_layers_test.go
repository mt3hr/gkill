package reps

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"

	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// audit_fields_all_layers_test.go は、記録の型ごとに作成・更新の欄8つ（アプリ・端末・利用者・時刻 × 作成/更新）を
// 全部違う値で書き、次のすべての読み取り口で同じ向きに戻ることを固定する。
//
//   - 実 rep（GetXxx / GetXxxHistories / FindXxx / FindKyous / GetKyou / GetKyouHistories と型固有の口）
//   - キャッシュ版（キャッシュ版へ追加した直後と、実 rep から UpdateCache で作り直した後）
//   - temp rep（GetXxxByTXID）と、CommitTx で実 rep へ確定した後
//
// SELECT の列順と rows.Scan の順、INSERT の列と引数の順がずれても SQLite はエラーを出さない。
// 通知・リポスト・Mi の temp rep・ファイルの記録の temp rep で作成者と作成端末の入れ替わりが続けて見つかったが、
// 既存のフィクスチャは作成と更新に同じ値を使い、端末と利用者も区別しないものが多く、どれも検出できていなかった。
// 通知とリポストは notification_repository_sqlite3_impl_test.go / re_kyou_repository_sqlite3_impl_test.go が固定する。
// SQL の書き方そのものは sql_column_order_scan_test.go がソースを走査して固定する。

// metaFields は作成・更新の欄8つ。
type metaFields struct {
	auditFields
	CreateTime, UpdateTime time.Time
}

// metaFieldsOf は任意のエンティティ（構造体かそのポインタ）から作成・更新の欄8つを reflect で抜く。
func metaFieldsOf(entity any) metaFields {
	v := reflect.Indirect(reflect.ValueOf(entity))
	getTime := func(name string) time.Time { return v.FieldByName(name).Interface().(time.Time) }
	return metaFields{
		auditFields: auditFieldsOf(entity),
		CreateTime:  getTime("CreateTime"),
		UpdateTime:  getTime("UpdateTime"),
	}
}

func assertMetaFieldsRoundTrip(t *testing.T, want, got metaFields) {
	t.Helper()
	assertAuditFieldsRoundTrip(t, want.auditFields, got.auditFields)
	if !got.CreateTime.Equal(want.CreateTime) || !got.UpdateTime.Equal(want.UpdateTime) {
		t.Errorf("作成・更新の時刻が往復で変わった: got create=%v update=%v want create=%v update=%v",
			got.CreateTime, got.UpdateTime, want.CreateTime, want.UpdateTime)
	}
}

// setDistinctMetaFields は entity（構造体へのポインタ）の作成・更新の欄8つを全部違う値にする。
// 時刻は秒単位（キャッシュ版は UNIX 秒で持つ）で、関連時刻（あれば）とも違う値にする。
func setDistinctMetaFields(entity any) {
	v := reflect.ValueOf(entity).Elem()
	base := testTime()
	setString := func(name, value string) { v.FieldByName(name).SetString(value) }
	setTime := func(name string, value time.Time) {
		if f := v.FieldByName(name); f.IsValid() {
			f.Set(reflect.ValueOf(value))
		}
	}
	setString("CreateApp", "create_app")
	setString("CreateDevice", "create_device")
	setString("CreateUser", "create_user")
	setString("UpdateApp", "update_app")
	setString("UpdateDevice", "update_device")
	setString("UpdateUser", "update_user")
	setTime("CreateTime", base)
	setTime("UpdateTime", base.Add(2*time.Hour))
	setTime("RelatedTime", base.Add(-3*time.Hour))
}

// metaRead は読み取り口1つ。返したエンティティのうち ID が一致するものを見る。
type metaRead struct {
	name string
	read func(ctx context.Context) ([]any, error)
}

func listAsAny[T any](list []T, err error) ([]any, error) {
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(list))
	for _, x := range list {
		out = append(out, x)
	}
	return out, nil
}

func oneAsAny[T any](x *T, err error) ([]any, error) {
	if err != nil || x == nil {
		return nil, err
	}
	return []any{*x}, nil
}

func kyouMapAsAny(kyous map[string][]Kyou, err error) ([]any, error) {
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, list := range kyous {
		for _, kyou := range list {
			out = append(out, kyou)
		}
	}
	return out, nil
}

// kyouMetaReads は Repository（Kyou を出す rep）共通の読み取り口3つ。
func kyouMetaReads(repo Repository, id string) []metaRead {
	return []metaRead{
		{"FindKyous", func(ctx context.Context) ([]any, error) {
			return kyouMapAsAny(repo.FindKyous(ctx, makeDefaultFindQuery()))
		}},
		{"GetKyou", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetKyou(ctx, id, nil)) }},
		{"GetKyouHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetKyouHistories(ctx, id)) }},
	}
}

func assertMetaFieldsOnReads(t *testing.T, want metaFields, id string, reads []metaRead) {
	t.Helper()
	ctx := context.Background()
	for _, r := range reads {
		t.Run(r.name, func(t *testing.T) {
			got, err := r.read(ctx)
			if err != nil {
				t.Fatalf("%s failed: %v", r.name, err)
			}
			matched := 0
			for _, entity := range got {
				if reflect.Indirect(reflect.ValueOf(entity)).FieldByName("ID").String() != id {
					continue
				}
				matched++
				assertMetaFieldsRoundTrip(t, want, metaFieldsOf(entity))
			}
			if matched == 0 {
				t.Fatalf("%s が追加した記録を返さない（%d 件返った）", r.name, len(got))
			}
		})
	}
}

// metaCase は1つの型について、各層の作り方・書き方・読み取り口をまとめたもの。
type metaCase[E any, R any] struct {
	// makeEntity は型固有の欄を埋めた記録を作る（作成・更新の欄は runMetaCase が上書きする）。
	makeEntity func() E
	newRepo    func(t *testing.T) R
	newCached  func(t *testing.T, base R) R
	add        func(ctx context.Context, repo R, e E) error
	reads      func(repo R, id string) []metaRead
	// stage は temp rep へ積む。stagedReads は temp rep の読み取り口。
	stage       func(ctx context.Context, temps *TempReps, e E, txID, userID, device string) error
	stagedReads func(temps *TempReps, txID, userID, device string) []metaRead
	// setWriteRep は CommitTx の書き込み先を GkillRepositories へ入れる。
	setWriteRep func(g *GkillRepositories, repo R)
}

func runMetaCase[E any, R any](t *testing.T, c metaCase[E, R]) {
	t.Helper()
	ctx := context.Background()
	newEntity := func() E {
		e := c.makeEntity()
		setDistinctMetaFields(&e)
		return e
	}
	entity := newEntity()
	want := metaFieldsOf(entity)
	id := reflect.ValueOf(entity).FieldByName("ID").String()

	t.Run("real", func(t *testing.T) {
		repo := c.newRepo(t)
		if err := c.add(ctx, repo, newEntity()); err != nil {
			t.Fatalf("add failed: %v", err)
		}
		assertMetaFieldsOnReads(t, want, id, c.reads(repo, id))
	})

	// 本番の書き込みと同じく、実 rep へ書いてからキャッシュ版へライトスルーする（WriteThroughXxxCache）。
	// キャッシュ版の AddXxxInfo はキャッシュ表にしか書かず、1件取得などを実 rep へ委譲する型もある。
	t.Run("cached", func(t *testing.T) {
		base := c.newRepo(t)
		repo := c.newCached(t, base)
		if err := c.add(ctx, base, newEntity()); err != nil {
			t.Fatalf("add to base failed: %v", err)
		}
		if err := c.add(ctx, repo, newEntity()); err != nil {
			t.Fatalf("write through to cache failed: %v", err)
		}
		assertMetaFieldsOnReads(t, want, id, c.reads(repo, id))
	})

	// 実 rep から読んでキャッシュへ入れ直す経路。読み取りと書き込みの両方で入れ替わっていると、
	// ここだけ2回の入れ替わりが打ち消し合って正しく見える（通知・リポストがそうだった）。
	t.Run("cached_rebuild", func(t *testing.T) {
		base := c.newRepo(t)
		if err := c.add(ctx, base, newEntity()); err != nil {
			t.Fatalf("add failed: %v", err)
		}
		repo := c.newCached(t, base)
		updater, ok := any(repo).(interface{ UpdateCache(context.Context) error })
		if !ok {
			t.Fatalf("キャッシュ版 %T が UpdateCache を持たない", repo)
		}
		if err := updater.UpdateCache(ctx); err != nil {
			t.Fatalf("UpdateCache failed: %v", err)
		}
		assertMetaFieldsOnReads(t, want, id, c.reads(repo, id))
	})

	t.Run("temp_and_commit", func(t *testing.T) {
		const txID, userID, device = "tx-meta-fields", "testuser", "testdevice"
		temps, err := NewTempReps(openSharedMemoryDB(t), &sync.RWMutex{})
		if err != nil {
			t.Fatalf("failed to create temp reps: %v", err)
		}
		if err := c.stage(ctx, temps, newEntity(), txID, userID, device); err != nil {
			t.Fatalf("stage failed: %v", err)
		}
		t.Run("staged", func(t *testing.T) {
			assertMetaFieldsOnReads(t, want, id, c.stagedReads(temps, txID, userID, device))
		})

		addressDAO, err := gkill_cache.NewLatestDataRepositoryAddressSQLite3Impl("testuser", openSharedMemoryDB(t), &sync.RWMutex{})
		if err != nil {
			t.Fatalf("failed to create latest data repository address dao: %v", err)
		}
		writeRep := c.newRepo(t)
		repositories := &GkillRepositories{TempReps: temps, LatestDataRepositoryAddressDAO: addressDAO}
		c.setWriteRep(repositories, writeRep)
		if _, err := repositories.CommitTx(ctx, txID, userID, device); err != nil {
			t.Fatalf("CommitTx failed: %v", err)
		}
		t.Run("committed", func(t *testing.T) {
			assertMetaFieldsOnReads(t, want, id, c.reads(writeRep, id))
		})
	})
}

// newCachedOver はキャッシュ版の構築関数を、インメモリのキャッシュ DB で呼ぶ。
func newCachedOver[R any](t *testing.T, base R, construct func(ctx context.Context, base R, cacheDB *sql.DB, m *sync.RWMutex, dbName string) (R, error)) R {
	t.Helper()
	ctx := context.Background()
	repo, err := construct(ctx, base, openMemoryDB(t), &sync.RWMutex{}, "META_FIELDS_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached repo: %v", err)
	}
	if closer, ok := any(repo).(interface{ Close(context.Context) error }); ok {
		t.Cleanup(func() { closer.Close(ctx) })
	}
	return repo
}

func TestMetaFieldsRoundTripOnAllLayers(t *testing.T) {
	t.Run("kmemo", func(t *testing.T) {
		runMetaCase(t, metaCase[Kmemo, KmemoRepository]{
			makeEntity: func() Kmemo { return makeKmemo("meta-kmemo", "作成・更新の欄の往復") },
			newRepo:    newTempKmemoRepo,
			newCached: func(t *testing.T, base KmemoRepository) KmemoRepository {
				return newCachedOver(t, base, NewKmemoRepositoryCachedSQLite3Impl)
			},
			add: func(ctx context.Context, repo KmemoRepository, e Kmemo) error { return repo.AddKmemoInfo(ctx, e) },
			reads: func(repo KmemoRepository, id string) []metaRead {
				return append(kyouMetaReads(repo, id),
					metaRead{"FindKmemo", func(ctx context.Context) ([]any, error) {
						return listAsAny(repo.FindKmemo(ctx, makeDefaultFindQuery()))
					}},
					metaRead{"GetKmemo", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetKmemo(ctx, id, nil)) }},
					metaRead{"GetKmemoHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetKmemoHistories(ctx, id)) }},
				)
			},
			stage: func(ctx context.Context, temps *TempReps, e Kmemo, txID, userID, device string) error {
				return temps.KmemoTempRep.AddKmemoInfo(ctx, e, txID, userID, device)
			},
			stagedReads: func(temps *TempReps, txID, userID, device string) []metaRead {
				return []metaRead{
					{"GetKmemosByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.KmemoTempRep.GetKmemosByTXID(ctx, txID, userID, device))
					}},
					// GetKyousByTXID は一時表に無い列を読むので必ず失敗する（呼び出し元なし。kmemo_temp_repository.go の doc）。
				}
			},
			setWriteRep: func(g *GkillRepositories, repo KmemoRepository) { g.WriteKmemoRep = repo },
		})
	})

	t.Run("kc", func(t *testing.T) {
		runMetaCase(t, metaCase[KC, KCRepository]{
			makeEntity: func() KC {
				return KC{ID: "meta-kc", Title: "作成・更新の欄の往復", NumValue: json.Number("1"), DataType: "kc"}
			},
			newRepo: newTempKCRepo,
			newCached: func(t *testing.T, base KCRepository) KCRepository {
				return newCachedOver(t, base, NewKCRepositoryCachedSQLite3Impl)
			},
			add: func(ctx context.Context, repo KCRepository, e KC) error { return repo.AddKCInfo(ctx, e) },
			reads: func(repo KCRepository, id string) []metaRead {
				return append(kyouMetaReads(repo, id),
					metaRead{"FindKC", func(ctx context.Context) ([]any, error) { return listAsAny(repo.FindKC(ctx, makeDefaultFindQuery())) }},
					metaRead{"GetKC", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetKC(ctx, id, nil)) }},
					metaRead{"GetKCHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetKCHistories(ctx, id)) }},
				)
			},
			stage: func(ctx context.Context, temps *TempReps, e KC, txID, userID, device string) error {
				return temps.KCTempRep.AddKCInfo(ctx, e, txID, userID, device)
			},
			stagedReads: func(temps *TempReps, txID, userID, device string) []metaRead {
				return []metaRead{
					{"GetKCsByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.KCTempRep.GetKCsByTXID(ctx, txID, userID, device))
					}},
					// GetKyousByTXID は一時表に無い列を読むので必ず失敗する（呼び出し元なし。kc_temp_repository.go の doc）。
				}
			},
			setWriteRep: func(g *GkillRepositories, repo KCRepository) { g.WriteKCRep = repo },
		})
	})

	t.Run("lantana", func(t *testing.T) {
		runMetaCase(t, metaCase[Lantana, LantanaRepository]{
			makeEntity: func() Lantana { return Lantana{ID: "meta-lantana", Mood: 5, DataType: "lantana"} },
			newRepo:    newTempLantanaRepo,
			newCached: func(t *testing.T, base LantanaRepository) LantanaRepository {
				return newCachedOver(t, base, NewLantanaRepositoryCachedSQLite3Impl)
			},
			add: func(ctx context.Context, repo LantanaRepository, e Lantana) error { return repo.AddLantanaInfo(ctx, e) },
			reads: func(repo LantanaRepository, id string) []metaRead {
				return append(kyouMetaReads(repo, id),
					metaRead{"FindLantana", func(ctx context.Context) ([]any, error) {
						return listAsAny(repo.FindLantana(ctx, makeDefaultFindQuery()))
					}},
					metaRead{"GetLantana", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetLantana(ctx, id, nil)) }},
					metaRead{"GetLantanaHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetLantanaHistories(ctx, id)) }},
				)
			},
			stage: func(ctx context.Context, temps *TempReps, e Lantana, txID, userID, device string) error {
				return temps.LantanaTempRep.AddLantanaInfo(ctx, e, txID, userID, device)
			},
			stagedReads: func(temps *TempReps, txID, userID, device string) []metaRead {
				return []metaRead{
					{"GetLantanasByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.LantanaTempRep.GetLantanasByTXID(ctx, txID, userID, device))
					}},
					// GetKyousByTXID は一時表に無い列を読むので必ず失敗する（呼び出し元なし。lantana_temp_repository.go の doc）。
				}
			},
			setWriteRep: func(g *GkillRepositories, repo LantanaRepository) { g.WriteLantanaRep = repo },
		})
	})

	t.Run("mi", func(t *testing.T) {
		runMetaCase(t, metaCase[Mi, MiRepository]{
			makeEntity: func() Mi { return makeMi("meta-mi", "作成・更新の欄の往復") },
			newRepo:    newTempMiRepo,
			newCached: func(t *testing.T, base MiRepository) MiRepository {
				return newCachedOver(t, base, NewMiRepositoryCachedSQLite3Impl)
			},
			add: func(ctx context.Context, repo MiRepository, e Mi) error { return repo.AddMiInfo(ctx, e) },
			reads: func(repo MiRepository, id string) []metaRead {
				return append(kyouMetaReads(repo, id),
					metaRead{"FindMi", func(ctx context.Context) ([]any, error) { return listAsAny(repo.FindMi(ctx, makeDefaultFindQuery())) }},
					metaRead{"GetMi", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetMi(ctx, id, nil)) }},
					metaRead{"GetMiHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetMiHistories(ctx, id)) }},
				)
			},
			stage: func(ctx context.Context, temps *TempReps, e Mi, txID, userID, device string) error {
				return temps.MiTempRep.AddMiInfo(ctx, e, txID, userID, device)
			},
			// Mi の temp rep の GetKyousByTXID は未対応（TestTempMi_GetKyousByTXIDIsUnsupported）。
			stagedReads: func(temps *TempReps, txID, userID, device string) []metaRead {
				return []metaRead{
					{"GetMisByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.MiTempRep.GetMisByTXID(ctx, txID, userID, device))
					}},
				}
			},
			setWriteRep: func(g *GkillRepositories, repo MiRepository) { g.WriteMiRep = repo },
		})
	})

	t.Run("mirekyou", func(t *testing.T) {
		runMetaCase(t, metaCase[MiReKyou, MiReKyouRepository]{
			makeEntity: func() MiReKyou { return makeMiReKyou("meta-mirekyou", "meta-mirekyou-target") },
			newRepo:    newTempMiReKyouRepo,
			newCached: func(t *testing.T, base MiReKyouRepository) MiReKyouRepository {
				return newCachedOver(t, base, func(ctx context.Context, base MiReKyouRepository, cacheDB *sql.DB, m *sync.RWMutex, dbName string) (MiReKyouRepository, error) {
					return NewMiReKyouRepositoryCachedSQLite3Impl(ctx, base, nil, cacheDB, m, dbName)
				})
			},
			add: func(ctx context.Context, repo MiReKyouRepository, e MiReKyou) error {
				return repo.AddMiReKyouInfo(ctx, e)
			},
			reads: func(repo MiReKyouRepository, id string) []metaRead {
				return append(kyouMetaReads(repo, id),
					metaRead{"FindMiReKyou", func(ctx context.Context) ([]any, error) {
						return listAsAny(repo.FindMiReKyou(ctx, makeDefaultFindQuery()))
					}},
					metaRead{"GetMiReKyou", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetMiReKyou(ctx, id, nil)) }},
					metaRead{"GetMiReKyouHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetMiReKyouHistories(ctx, id)) }},
					metaRead{"GetMiReKyousAllLatest", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetMiReKyousAllLatest(ctx)) }},
				)
			},
			stage: func(ctx context.Context, temps *TempReps, e MiReKyou, txID, userID, device string) error {
				return temps.MiReKyouTempRep.AddMiReKyouInfo(ctx, e, txID, userID, device)
			},
			stagedReads: func(temps *TempReps, txID, userID, device string) []metaRead {
				return []metaRead{
					{"GetMiReKyousByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.MiReKyouTempRep.GetMiReKyousByTXID(ctx, txID, userID, device))
					}},
					{"GetKyousByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.MiReKyouTempRep.GetKyousByTXID(ctx, txID, userID, device))
					}},
				}
			},
			setWriteRep: func(g *GkillRepositories, repo MiReKyouRepository) { g.WriteMiReKyouRep = repo },
		})
	})

	t.Run("nlog", func(t *testing.T) {
		runMetaCase(t, metaCase[Nlog, NlogRepository]{
			makeEntity: func() Nlog {
				return Nlog{ID: "meta-nlog", Shop: "ShopA", Title: "作成・更新の欄の往復", Amount: json.Number("-1000"), DataType: "nlog"}
			},
			newRepo: newTempNlogRepo,
			newCached: func(t *testing.T, base NlogRepository) NlogRepository {
				return newCachedOver(t, base, NewNlogRepositoryCachedSQLite3Impl)
			},
			add: func(ctx context.Context, repo NlogRepository, e Nlog) error { return repo.AddNlogInfo(ctx, e) },
			reads: func(repo NlogRepository, id string) []metaRead {
				return append(kyouMetaReads(repo, id),
					metaRead{"FindNlog", func(ctx context.Context) ([]any, error) { return listAsAny(repo.FindNlog(ctx, makeDefaultFindQuery())) }},
					metaRead{"GetNlog", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetNlog(ctx, id, nil)) }},
					metaRead{"GetNlogHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetNlogHistories(ctx, id)) }},
				)
			},
			stage: func(ctx context.Context, temps *TempReps, e Nlog, txID, userID, device string) error {
				return temps.NlogTempRep.AddNlogInfo(ctx, e, txID, userID, device)
			},
			stagedReads: func(temps *TempReps, txID, userID, device string) []metaRead {
				return []metaRead{
					{"GetNlogsByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.NlogTempRep.GetNlogsByTXID(ctx, txID, userID, device))
					}},
					// GetKyousByTXID は一時表に無い列を読むので必ず失敗する（呼び出し元なし。nlog_temp_repository.go の doc）。
				}
			},
			setWriteRep: func(g *GkillRepositories, repo NlogRepository) { g.WriteNlogRep = repo },
		})
	})

	t.Run("timeis", func(t *testing.T) {
		runMetaCase(t, metaCase[TimeIs, TimeIsRepository]{
			makeEntity: func() TimeIs {
				return TimeIs{ID: "meta-timeis", Title: "作成・更新の欄の往復", StartTime: testTime().Add(-3 * time.Hour), DataType: "timeis"}
			},
			newRepo: newTempTimeIsRepo,
			newCached: func(t *testing.T, base TimeIsRepository) TimeIsRepository {
				return newCachedOver(t, base, NewTimeIsRepositoryCachedSQLite3Impl)
			},
			add: func(ctx context.Context, repo TimeIsRepository, e TimeIs) error { return repo.AddTimeIsInfo(ctx, e) },
			reads: func(repo TimeIsRepository, id string) []metaRead {
				return append(kyouMetaReads(repo, id),
					metaRead{"FindTimeIs", func(ctx context.Context) ([]any, error) {
						return listAsAny(repo.FindTimeIs(ctx, makeDefaultFindQuery()))
					}},
					metaRead{"GetTimeIs", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetTimeIs(ctx, id, nil)) }},
					metaRead{"GetTimeIsHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetTimeIsHistories(ctx, id)) }},
				)
			},
			stage: func(ctx context.Context, temps *TempReps, e TimeIs, txID, userID, device string) error {
				return temps.TimeIsTempRep.AddTimeIsInfo(ctx, e, txID, userID, device)
			},
			stagedReads: func(temps *TempReps, txID, userID, device string) []metaRead {
				return []metaRead{
					{"GetTimeIssByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.TimeIsTempRep.GetTimeIssByTXID(ctx, txID, userID, device))
					}},
					// GetKyousByTXID は一時表に無い列を読むので必ず失敗する（呼び出し元なし。time_is_temp_repository.go の doc）。
				}
			},
			setWriteRep: func(g *GkillRepositories, repo TimeIsRepository) { g.WriteTimeIsRep = repo },
		})
	})

	t.Run("urlog", func(t *testing.T) {
		runMetaCase(t, metaCase[URLog, URLogRepository]{
			makeEntity: func() URLog {
				return URLog{ID: "meta-urlog", URL: "https://example.com/meta", Title: "作成・更新の欄の往復", DataType: "urlog"}
			},
			newRepo: newTempURLogRepo,
			newCached: func(t *testing.T, base URLogRepository) URLogRepository {
				return newCachedOver(t, base, NewURLogRepositoryCachedSQLite3Impl)
			},
			add: func(ctx context.Context, repo URLogRepository, e URLog) error { return repo.AddURLogInfo(ctx, e) },
			reads: func(repo URLogRepository, id string) []metaRead {
				return append(kyouMetaReads(repo, id),
					metaRead{"FindURLog", func(ctx context.Context) ([]any, error) {
						return listAsAny(repo.FindURLog(ctx, makeDefaultFindQuery()))
					}},
					// キャッシュ版はサムネイルを除く指定のときだけキャッシュ表を読む（それ以外は実 rep へ委譲）。
					metaRead{"FindURLog_ExcludeThumbnail", func(ctx context.Context) ([]any, error) {
						query := makeDefaultFindQuery()
						query.ExcludeURLogThumbnailImage = true
						return listAsAny(repo.FindURLog(ctx, query))
					}},
					metaRead{"GetURLog", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetURLog(ctx, id, nil)) }},
					metaRead{"GetURLogHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetURLogHistories(ctx, id)) }},
				)
			},
			stage: func(ctx context.Context, temps *TempReps, e URLog, txID, userID, device string) error {
				return temps.URLogTempRep.AddURLogInfo(ctx, e, txID, userID, device)
			},
			stagedReads: func(temps *TempReps, txID, userID, device string) []metaRead {
				return []metaRead{
					{"GetURLogsByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.URLogTempRep.GetURLogsByTXID(ctx, txID, userID, device))
					}},
					// GetKyousByTXID は一時表に無い列を読むので必ず失敗する（呼び出し元なし。ur_log_temp_repository.go の doc）。
				}
			},
			setWriteRep: func(g *GkillRepositories, repo URLogRepository) { g.WriteURLogRep = repo },
		})
	})

	// ファイルの記録の temp rep は INSERT の列（CREATE_USER, CREATE_DEVICE）と引数（CreateDevice, CreateUser）が逆で、
	// 連鎖削除（必ず temp rep → CommitTx を通る）のたびに作成者と作成端末が入れ替わって確定していた。
	t.Run("idf", func(t *testing.T) {
		runMetaCase(t, metaCase[IDFKyou, IDFKyouRepository]{
			makeEntity: func() IDFKyou {
				// キャッシュ版は CONTENT_PATH が空の行を読み飛ばすので、置き場の絶対パスに見える値を入れておく。
				return IDFKyou{ID: "meta-idf", TargetFile: "meta.txt", ContentPath: "/meta/meta.txt", DataType: "idf"}
			},
			newRepo: newTempIDFKyouRepo,
			newCached: func(t *testing.T, base IDFKyouRepository) IDFKyouRepository {
				return newCachedOver(t, base, NewIDFCachedRep)
			},
			add: func(ctx context.Context, repo IDFKyouRepository, e IDFKyou) error { return repo.AddIDFKyouInfo(ctx, e) },
			reads: func(repo IDFKyouRepository, id string) []metaRead {
				return append(kyouMetaReads(repo, id),
					metaRead{"FindIDFKyou", func(ctx context.Context) ([]any, error) {
						return listAsAny(repo.FindIDFKyou(ctx, makeDefaultFindQuery()))
					}},
					metaRead{"GetIDFKyou", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetIDFKyou(ctx, id, nil)) }},
					metaRead{"GetIDFKyouHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetIDFKyouHistories(ctx, id)) }},
					metaRead{"GetIDFKyouByTargetFile", func(ctx context.Context) ([]any, error) {
						return oneAsAny(repo.GetIDFKyouByTargetFile(ctx, "meta.txt"))
					}},
				)
			},
			stage: func(ctx context.Context, temps *TempReps, e IDFKyou, txID, userID, device string) error {
				return temps.IDFKyouTempRep.AddIDFKyouInfo(ctx, e, txID, userID, device)
			},
			stagedReads: func(temps *TempReps, txID, userID, device string) []metaRead {
				return []metaRead{
					{"GetIDFKyousByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.IDFKyouTempRep.GetIDFKyousByTXID(ctx, txID, userID, device))
					}},
					{"GetKyousByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.IDFKyouTempRep.GetKyousByTXID(ctx, txID, userID, device))
					}},
				}
			},
			setWriteRep: func(g *GkillRepositories, repo IDFKyouRepository) { g.WriteIDFKyouRep = repo },
		})
	})

	t.Run("tag", func(t *testing.T) {
		runMetaCase(t, metaCase[Tag, TagRepository]{
			makeEntity: func() Tag { return makeTag("meta-tag", "meta-tag-target", "TagA") },
			newRepo:    newTempTagRepo,
			newCached: func(t *testing.T, base TagRepository) TagRepository {
				return newCachedOver(t, base, NewTagRepositoryCachedSQLite3Impl)
			},
			add: func(ctx context.Context, repo TagRepository, e Tag) error { return repo.AddTagInfo(ctx, e) },
			reads: func(repo TagRepository, id string) []metaRead {
				return []metaRead{
					{"FindTags", func(ctx context.Context) ([]any, error) { return listAsAny(repo.FindTags(ctx, makeDefaultFindQuery())) }},
					{"GetTag", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetTag(ctx, id, nil)) }},
					{"GetTagHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetTagHistories(ctx, id)) }},
					{"GetTagsByTagName", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetTagsByTagName(ctx, "TagA")) }},
					{"GetTagsByTargetID", func(ctx context.Context) ([]any, error) {
						return listAsAny(repo.GetTagsByTargetID(ctx, "meta-tag-target"))
					}},
					{"GetAllTags", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetAllTags(ctx)) }},
				}
			},
			stage: func(ctx context.Context, temps *TempReps, e Tag, txID, userID, device string) error {
				return temps.TagTempRep.AddTagInfo(ctx, e, txID, userID, device)
			},
			stagedReads: func(temps *TempReps, txID, userID, device string) []metaRead {
				return []metaRead{
					{"GetTagsByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.TagTempRep.GetTagsByTXID(ctx, txID, userID, device))
					}},
				}
			},
			setWriteRep: func(g *GkillRepositories, repo TagRepository) { g.WriteTagRep = repo },
		})
	})

	t.Run("text", func(t *testing.T) {
		runMetaCase(t, metaCase[Text, TextRepository]{
			makeEntity: func() Text { return makeText("meta-text", "meta-text-target", "作成・更新の欄の往復") },
			newRepo:    newTempTextRepo,
			newCached: func(t *testing.T, base TextRepository) TextRepository {
				return newCachedOver(t, base, NewTextRepositoryCachedSQLite3Impl)
			},
			add: func(ctx context.Context, repo TextRepository, e Text) error { return repo.AddTextInfo(ctx, e) },
			reads: func(repo TextRepository, id string) []metaRead {
				return []metaRead{
					{"FindTexts", func(ctx context.Context) ([]any, error) {
						return listAsAny(repo.FindTexts(ctx, makeDefaultFindQuery()))
					}},
					{"GetText", func(ctx context.Context) ([]any, error) { return oneAsAny(repo.GetText(ctx, id, nil)) }},
					{"GetTextHistories", func(ctx context.Context) ([]any, error) { return listAsAny(repo.GetTextHistories(ctx, id)) }},
					{"GetTextsByTargetID", func(ctx context.Context) ([]any, error) {
						return listAsAny(repo.GetTextsByTargetID(ctx, "meta-text-target"))
					}},
				}
			},
			stage: func(ctx context.Context, temps *TempReps, e Text, txID, userID, device string) error {
				return temps.TextTempRep.AddTextInfo(ctx, e, txID, userID, device)
			},
			stagedReads: func(temps *TempReps, txID, userID, device string) []metaRead {
				return []metaRead{
					{"GetTextsByTXID", func(ctx context.Context) ([]any, error) {
						return listAsAny(temps.TextTempRep.GetTextsByTXID(ctx, txID, userID, device))
					}},
				}
			},
			setWriteRep: func(g *GkillRepositories, repo TextRepository) { g.WriteTextRep = repo },
		})
	})
}

// TestSetDistinctMetaFieldsMakesEveryFieldDifferent は、上の往復テストの前提（8欄と関連時刻が全部違う値）を固定する。
// 値が重なると、その2欄の入れ替わりは往復テストで検出できない。
func TestSetDistinctMetaFieldsMakesEveryFieldDifferent(t *testing.T) {
	k := Kmemo{}
	setDistinctMetaFields(&k)
	strs := []string{k.CreateApp, k.CreateDevice, k.CreateUser, k.UpdateApp, k.UpdateDevice, k.UpdateUser}
	seen := map[string]bool{}
	for _, s := range strs {
		if seen[s] {
			t.Fatalf("作成・更新の文字列の欄に同じ値がある: %v", strs)
		}
		seen[s] = true
	}
	times := []time.Time{k.CreateTime, k.UpdateTime, k.RelatedTime}
	for i := range times {
		for j := range times {
			if i != j && times[i].Equal(times[j]) {
				t.Fatalf("作成・更新・関連の時刻に同じ値がある: %v", times)
			}
		}
	}
	if k.CreateTime.Nanosecond() != 0 {
		t.Fatalf("時刻は秒単位にする（キャッシュ版は UNIX 秒で持つ）: %v", k.CreateTime)
	}
}
