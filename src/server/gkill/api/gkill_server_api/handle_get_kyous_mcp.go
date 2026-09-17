package gkill_server_api

// 複合カーソル(時刻+ID)・厳密なLimit/MaxSizeMB・count_only/group_by の契約:
// documents/adr/0604-mcp-composite-cursor-strict-limits.md
// (旧: 時刻のみカーソルのため Limit を厳密に守れなかった — ADR-0603、Superseded)

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleGetKyousMCP は、MCPサーバ向けにKyouを検索し、型ごとのペイロードと
// 付随データ (タグ・テキスト・通知) を1件にまとめたDTOをページング・集計して返します。
//
// POST /api/get_kyous_mcp（wrapNoAuth）
// req_res.GetKyousMCPRequest / req_res.GetKyousMCPResponse
//
// wrapNoAuth登録ですが、ハンドラ内でSessionIDからアカウントを解決するので未認証では使えません。
// Limitは1〜1000にクランプ (未指定は50)、MaxSizeMBの未指定は1.0、
// Queryはnilなら空のクエリに差し替え、いずれの場合も OnlyLatestData = true に上書きします。
//
// v2 の契約（詳細と却下案: documents/adr/0604-mcp-composite-cursor-strict-limits.md）:
//   - 並び順は (RelatedTime降順, ID昇順) の全順序。カーソルは複合形式 "{RFC3339Nano}::{ID}" で、
//     同一時刻のかたまりの途中からでも再開できるため **Limit と MaxSizeMB は厳密な上限**です
//     （唯一の例外はページ先頭の1件が単独で MaxSizeMB を超えるときで、そのまま返して警告します。
//     返さないと0件+has_more=trueの永久ループになるため）。
//   - 旧形式カーソル（RFC3339単独・日付のみ）も受理します。解釈できないカーソルはエラーです
//     （黙って1ページ目に戻すとページングが終わらないため）。
//   - カーソルはクエリの期間上限(CalendarEndDate)へ押し下げるため、2ページ目以降のハンドラは
//     全件数を知りません。TotalCount は cursor 無しの応答にのみ入り、全応答に RemainingCount が入ります。
//   - CountOnly はDTO構築・付随データ取得を全て飛ばして件数だけを返します。
//     GroupBy は buckets へのバケット集計を返します。どちらも cursor とは併用できません（エラー）。
//   - DataTypes / NumMin / NumMax / IDFKinds はリクエストレベルの絞り込みで、
//     FindQuery の検索結果に対して件数・ページングより前に適用されます。
//   - 未知のフィルタ値（rep_types / tags / reps / data_types の綴り違い等）は Warnings で指摘します。
//     エラーにはしません（「実在するが該当0件」の正当な経路を壊さないため）。
//
// URLogのサムネイル画像はAIクライアントで扱えないうえ巨大なので、DBから読む段階で外します。
// IDFペイロードのFilePathはローカルリクエストのときだけ、FileSizeはIncludeFileSizeのときだけ入ります。
// Kyouのid/rep_nameは常時入ります（旧v1の要求フラグは廃止。追撃クエリの前提のため）。
// ペイロードの分岐はDataTypeそのものではなく payloadKindOfDataType で寄せた種別で行います。
// Mi/MiReKyou/TimeIsのDataTypeは射影ごとに枝分かれする(mi_create、mirekyou_limit、
// timeis_start ...)ので、素の型名との完全一致では拾えません。
// そこにも当てはまらないKyouはプラグイン由来とみなし、rep_nameでプラグインを
// 引き当てて、本文の代わりにrep_name/kyou_idを載せたペイロードを返します
// （本文はgkill側に保存されておらず、別途コンテンツHTML取得が要るため）。
func (g *GkillServerAPI) HandleGetKyousMCP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKyousMCPRequest{}
	response := &req_res.GetKyousMCPResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close request body", "error", fmt.Sprintf("%q", err))
		}
	}()
	defer func() {
		writeErrorStatus(r.Context(), w, response.Errors)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyous mcp response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse get kyous mcp response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKyousMCPResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyous mcp request from json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse get kyous mcp request from json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKyousMCPRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// デフォルト設定
	const maxLimit = 1000
	if request.Limit <= 0 {
		request.Limit = 50
	} else if request.Limit > maxLimit {
		request.Limit = maxLimit
	}
	if request.MaxSizeMB <= 0 {
		request.MaxSizeMB = 1.0
	}
	if request.Query == nil {
		request.Query = &find.FindQuery{}
	}
	request.Query.OnlyLatestData = true

	// カーソル押し下げ**前**の検索条件を控える。cursor 頁で Mi / MiReKyou の代表射影を
	// 「1頁目と同じ窓」で選び直すために使う（revalidateMiEntriesAgainstOriginalWindow）。
	// 浅いコピーでよい —— 押し下げは CalendarEndDate のポインタを差し替えるだけで、指す先は書き換えない。
	originalQuery := *request.Query

	// カーソルをクエリの期間上限へ押し下げる。
	//
	// ★これが無いと、ページ1枚(最大1000件)を返すためだけに毎回全期間を検索し直す。
	//   実データ(30年・56万件・376リポジトリ)では1リクエストあたり +1.8GB のメモリと
	//   数十分を要し、それがページ数(約568回)ぶん繰り返されてサーバが膨れ続ける
	//   (2026-08-16 実測)。ページングがサーバ側の仕事をまったく軽くしていなかった。
	//
	//   CalendarEndDate は RelatedTime の上限(境界を含む)としてSQLまで降りるので、
	//   ここへ落とせば検索対象そのものが「カーソル以降」に縮む。
	//   境界ちょうど(同一時刻)の件はこのあとのカーソル走査が従来どおり読み飛ばすため、
	//   返る中身は押し下げの前後で変わらない。
	//   呼び出し元が期間を指定している場合は狭いほうを採る。
	cursor := mcpCursor{}
	hasCursor := false
	if request.Cursor != "" {
		parsedCursor, parseErr := parseMCPCursor(request.Cursor)
		if parseErr != nil {
			// ★解釈できないカーソルを黙って無視してはいけない。
			//   以前は無視して1ページ目を返していたため、呼び出し側は同じページを
			//   受け取り続け、ページングが永久に終わらなかった。
			err = fmt.Errorf("error at parse cursor: %w", parseErr)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse cursor", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKyousMCPRequestDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
				Cause:        parseErr,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		cursor = parsedCursor
		hasCursor = true
		if request.Query.CalendarEndDate == nil || request.Query.CalendarEndDate.After(cursor.time) {
			request.Query.CalendarEndDate = &cursor.time
		}
	}

	// count_only / group_by は「条件に合う全件」を数える口なので cursor と併用できない。
	// 黙って片方を無視すると呼び出し側が気付けない（不正カーソル黙殺と同じ罠）ためエラーにする。
	if hasCursor && (request.CountOnly || request.GroupBy != "") {
		err = fmt.Errorf("error at get kyous mcp: count_only/group_by cannot be combined with cursor")
		slog.Log(r.Context(), gkill_log.Debug, "error at get kyous mcp", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.InvalidGetKyousMCPRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
		})
		return
	}
	// count_only と group_by の併用もエラー。group_by は既に「件数だけ（buckets + total_count）」を
	// 返す口なので、count_only を重ねる意味は無く、以前は count_only の早期 return が group_by を
	// 黙って捨てていた（buckets が無い応答。2026-09-18 の実利用報告）。理由つきの文言は MCP 層が先に出す。
	if request.CountOnly && request.GroupBy != "" {
		err = fmt.Errorf("error at get kyous mcp: count_only cannot be combined with group_by")
		slog.Log(r.Context(), gkill_log.Debug, "error at get kyous mcp", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.InvalidGetKyousMCPRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
		})
		return
	}
	if request.GroupBy != "" && !isValidMCPGroupBy(request.GroupBy) {
		err = fmt.Errorf("error at get kyous mcp: unknown group_by %q", request.GroupBy)
		slog.Log(r.Context(), gkill_log.Debug, "error at get kyous mcp", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.InvalidGetKyousMCPRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
		})
		return
	}

	// アカウントを取得
	// wrapNoAuth なので認証は自前。3段（アカウント→端末→リポジトリ）は
	// wrapAuthRepos と同じ処理なので共通ヘルパへ寄せてある。
	userID, device, repositories, gkillError := g.resolveSelfAuthContext(
		r.Context(), request.SessionID, request.LocaleName, "FAILED_GET_KYOUS_MESSAGE")
	if gkillError != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// プラグイン検索の失敗を警告として回収する。
	//
	// この仕込みは長らく usecase.GetKyous（/api/get_kyous。Web が使う）にしか無く、
	// MCP は FindKyous を素で呼んでいたため、**プラグイン検索がコケても MCP には
	// 何も伝わらなかった**。実利用のレビューで「is_alive:true なのに0件」のプラグインを
	// 前に足止めされている（2026-08-25）。エラーではなく警告なのは、
	// errors へ載せると呼び出し側が検索全体を失敗扱いにして結果を捨てるため。
	findCtx := reps.WithFindWarnings(r.Context())

	// Kyou一覧を取得
	allKyous, gkillErrors, err := g.FindFilter.FindKyous(findCtx, userID, device, g.GkillDAOManager, request.Query)
	if len(gkillErrors) != 0 || err != nil {
		if err != nil {
			err = fmt.Errorf("error at find kyous mcp: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at find kyous mcp", "error", fmt.Sprintf("%q", err))
			// 検索が失敗したのにGkillErrorが1つも無いことがある(repのSQLエラーなど)。
			// そのまま返すと errors:null + 0件 になり、呼び出し側からは
			// 「成功・該当0件」と区別が付かない。理由はEnsureNotEmptyのコメント。
			gkillErrors = message.EnsureNotEmpty(gkillErrors, message.FindKyousError,
				api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}), err)
		}
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	// リクエストレベルの絞り込み（FindQueryではなく検索結果に対して掛かる）。
	// 件数(total_count/count_only/group_by)・ページングより前に適用する。
	reportRequestFilterError := func(what string, err error) {
		err = fmt.Errorf("error at apply %s for get kyous mcp: %w", what, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at apply", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.FindKyousError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
		})
	}
	// query.ids の不一致は、絞り込みの前の結果と突き合わせる（絞り込みで落ちた ID を
	// 「一致なし」と言わないため）。他の未知値警告と同じく count_only でも出る。
	response.Warnings = append(response.Warnings, collectMCPUnmatchedIDWarnings(request.Query, allKyous)...)
	// data_types のエンティティ名（timeis / mi / mirekyou）は射影名へ展開してから照合する。
	// 警告側（collectMCPUnknownValueWarnings）には展開前の値を渡す（expandMCPDataTypes のコメント）。
	allKyous = applyMCPDataTypesFilter(allKyous, expandMCPDataTypes(request.DataTypes))
	allKyous = applyMCPCreateAppsFilter(allKyous, request.CreateApps)
	allKyous = applyMCPUpdateAppsFilter(allKyous, request.UpdateApps)
	if request.NumMin != nil || request.NumMax != nil {
		filtered, filterWarnings, filterErr := applyMCPNumFilter(r.Context(), repositories, allKyous, request.NumMin, request.NumMax)
		response.Warnings = append(response.Warnings, filterWarnings...)
		if filterErr != nil {
			reportRequestFilterError("num filter", filterErr)
			return
		}
		allKyous = filtered
	}
	if request.IDFKinds != nil {
		filtered, filterWarnings, filterErr := applyMCPIDFKindsFilter(r.Context(), repositories, allKyous, request.IDFKinds)
		response.Warnings = append(response.Warnings, filterWarnings...)
		if filterErr != nil {
			reportRequestFilterError("idf_kinds filter", filterErr)
			return
		}
		allKyous = filtered
	}

	// 未知のフィルタ値の警告（綴り違いが黙って0件になるのを防ぐ。外部監査 S7）。
	// count_only でも実施する — 件数確認こそタイポ検索の入口のため。
	for _, pluginName := range reps.PluginFindWarnings(findCtx) {
		response.Warnings = append(response.Warnings,
			fmt.Sprintf("plugin %q failed during this search, so its records are missing from the result. "+
				"Check gkill_get_plugin_list (is_alive / has_last_error / typed_index)", pluginName))
	}
	// 読み込めなかったrepの警告。
	// このrepは存在しないので、名前を query.reps へ渡してはいけない
	// （非空のrep名は「実在するが選ばれていない」と扱われ、エラーも警告も無く0件になる）。
	for _, repName := range reps.RepLoadWarnings(findCtx) {
		response.Warnings = append(response.Warnings,
			fmt.Sprintf("repository %q could not be loaded, so its records are missing from the result. "+
				"Do not pass this name in query.reps; it would silently match nothing", repName))
	}
	response.Warnings = append(response.Warnings, collectMCPUnknownValueWarnings(r.Context(), repositories, request.Query, request.DataTypes)...)

	// (RelatedTime降順, ID昇順) の全順序ソート。
	// IDのタイブレークは複合カーソルの前提（同一時刻のかたまりの中で位置を特定できる）。
	slices.SortFunc(allKyous, func(a, b reps.Kyou) int {
		if c := b.RelatedTime.Compare(a.RelatedTime); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})

	totalCount := len(allKyous)

	// count_only: DTO構築・typed取得・付随データ3N・TimeIs全ロードを全て飛ばして件数だけ返す。
	// 旧v1は「limit:1で総数だけ読む」が最安の裏技だったが、それでも1件ぶんのフルDTOと
	// 使われないファイルURLトークンが毎回付いてきた（外部監査 B1）。
	if request.CountOnly {
		response.TotalCount = &totalCount
		appendGetKyousMCPSuccess(response, request.LocaleName)
		return
	}

	// group_by: バケット集計を返す。limit / max_size_mb は適用されない
	// （バケット数はmcpBucketLimitで有界）。
	if request.GroupBy != "" {
		buckets, bucketWarnings, bucketErr := bucketizeMCPKyous(r.Context(), repositories, allKyous, request.GroupBy)
		response.Warnings = append(response.Warnings, bucketWarnings...)
		if bucketErr != nil {
			reportRequestFilterError("group_by", bucketErr)
			return
		}
		response.Buckets = buckets
		response.TotalCount = &totalCount
		appendGetKyousMCPSuccess(response, request.LocaleName)
		return
	}

	// カーソル適用。
	// 期間上限へ押し下げ済みなので、ここで読み飛ばすのは境界(カーソルと同一時刻)ぶんだけ。
	// 複合カーソル（時刻+ID）なら同一時刻のかたまりの内側でも「IDがカーソルより後」から
	// 正確に再開できる。旧形式（時刻のみ）は従来どおり「厳密に前」から。
	startIdx := 0
	if hasCursor {
		found := false
		for i, kyou := range allKyous {
			if kyou.RelatedTime.Before(cursor.time) {
				startIdx = i
				found = true
				break
			}
			if cursor.hasID && kyou.RelatedTime.Equal(cursor.time) && kyou.ID > cursor.id {
				startIdx = i
				found = true
				break
			}
		}
		if !found {
			startIdx = len(allKyous)
		}
	}

	batch := allKyous[startIdx:]

	// ★cursor 頁では Mi / MiReKyou の entry を元の窓で選び直す。
	//   押し下げた窓では Mi の代表射影が変わり（mi_check の時刻は UPDATE_TIME なので窓外へ落ち、
	//   CREATE_TIME の mi_create が代表になる）、1頁目に返した記録がカーソルより後ろに
	//   別の射影名で再出現していた。remaining_count が減らない（145→145→143）のはその副作用で、
	//   実害は同じ記録がページをまたいで重複すること（2026-09-18 の実利用報告。ADR-0621）。
	if hasCursor {
		revalidated, revalidateWarnings := g.revalidateMiEntriesAgainstOriginalWindow(r.Context(), userID, device, &originalQuery, batch)
		response.Warnings = append(response.Warnings, revalidateWarnings...)
		batch = revalidated
	}

	// 候補IDを収集。
	// ★v2ではLimitは厳密な上限。複合カーソルが同一時刻のかたまりの途中からでも
	//   再開できるため、旧v1の「かたまりの終わりまで伸ばす」延長は不要になった
	//   （ADR-0604。旧v1の事情はADR-0603を参照）。
	// request.Limitは冒頭で[1,maxLimit]にクランプ済み。ここでは batch のインデックス範囲
	// (batch[i])を安全にするため candidateCount <= len(batch) にクランプする。
	candidateCount := request.Limit
	if candidateCount < 0 {
		candidateCount = 0
	}
	if candidateCount > len(batch) {
		candidateCount = len(batch)
	}
	if candidateCount > maxLimit {
		candidateCount = maxLimit
	}
	candidateIDs := make([]string, 0)
	for i := range candidateCount {
		candidateIDs = append(candidateIDs, batch[i].ID)
	}

	// 候補ID用クエリを作成
	findQueryForBatch := &find.FindQuery{
		IDs:            candidateIDs,
		OnlyLatestData: true,
	}

	// 各型の詳細データを一括取得してマップを構築する。
	//
	// 失敗を握りつぶしてはいけない。握りつぶすと、その型のペイロードだけが欠けたまま
	// HTTP 200 + errors:null で返り、呼び出し側からは「その型の記録が無かった」と区別が付かない。
	// 同じ11型のファンアウトを handle_get_shared_kyous.go も1件ずつエラーにして返している
	reportFindDetailError := func(dataType string, err error) {
		err = fmt.Errorf("error at find %s for get kyous mcp user id = %s device = %s: %w", dataType, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at find", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.FindKyousError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
		})
	}

	kmemos, err := repositories.KmemoReps.FindKmemo(r.Context(), findQueryForBatch)
	if err != nil {
		reportFindDetailError("kmemo", err)
		return
	}
	kmemoMap := make(map[string]reps.Kmemo, len(kmemos))
	for _, k := range kmemos {
		kmemoMap[k.ID] = k
	}

	kcs, err := repositories.KCReps.FindKC(r.Context(), findQueryForBatch)
	if err != nil {
		reportFindDetailError("kc", err)
		return
	}
	kcMap := make(map[string]reps.KC, len(kcs))
	for _, k := range kcs {
		kcMap[k.ID] = k
	}

	timeiss, err := repositories.TimeIsReps.FindTimeIs(r.Context(), findQueryForBatch)
	if err != nil {
		reportFindDetailError("timeis", err)
		return
	}
	timeIsMap := make(map[string]reps.TimeIs, len(timeiss))
	for _, t := range timeiss {
		timeIsMap[t.ID] = t
	}

	nlogs, err := repositories.NlogReps.FindNlog(r.Context(), findQueryForBatch)
	if err != nil {
		reportFindDetailError("nlog", err)
		return
	}
	nlogMap := make(map[string]reps.Nlog, len(nlogs))
	for _, n := range nlogs {
		nlogMap[n.ID] = n
	}

	lantanas, err := repositories.LantanaReps.FindLantana(r.Context(), findQueryForBatch)
	if err != nil {
		reportFindDetailError("lantana", err)
		return
	}
	lantanaMap := make(map[string]reps.Lantana, len(lantanas))
	for _, l := range lantanas {
		lantanaMap[l.ID] = l
	}

	// URLogのサムネイルはbase64画像で、実データでは227行で90MBある(1行最大10MB)。
	// MCPの利用者はAIクライアントで画像本体を使えないため、DBから読む段階で外す。
	findQueryForURLog := *findQueryForBatch
	findQueryForURLog.ExcludeURLogThumbnailImage = true
	urlogs, err := repositories.URLogReps.FindURLog(r.Context(), &findQueryForURLog)
	if err != nil {
		reportFindDetailError("urlog", err)
		return
	}
	urlogMap := make(map[string]reps.URLog, len(urlogs))
	for _, u := range urlogs {
		urlogMap[u.ID] = u
	}

	idfKyous, err := repositories.IDFKyouReps.FindIDFKyou(r.Context(), findQueryForBatch)
	if err != nil {
		reportFindDetailError("idf_kyou", err)
		return
	}
	idfKyouMap := make(map[string]reps.IDFKyou, len(idfKyous))
	for _, idfk := range idfKyous {
		idfKyouMap[idfk.ID] = idfk
	}

	gitCommitLogs, err := repositories.GitCommitLogReps.FindGitCommitLog(r.Context(), findQueryForBatch)
	if err != nil {
		reportFindDetailError("git_commit_log", err)
		return
	}
	gitCommitLogMap := make(map[string]reps.GitCommitLog, len(gitCommitLogs))
	for _, gcl := range gitCommitLogs {
		gitCommitLogMap[gcl.ID] = gcl
	}

	// Mi / MiReKyou のFindは、5つのIncludeXxxMiがどのSQL射影を流すかのスイッチになっており、
	// 全falseだとUNIONの元が1本も無くなって必ず空を返す(mi_repository_sqlite3_impl.goの
	// len(sqlSegments)==0、mi_re_kyou_sql.goのbuildMiReKyouSQLも同じ)。
	// ここはID指定でペイロードの元データを引くだけなので、
	// 全行に当たる作成射影(CREATE_TIME IS NOT NULL)だけを立てる。
	findQueryForMi := *findQueryForBatch
	findQueryForMi.IncludeCreateMi = true

	mis, err := repositories.MiReps.FindMi(r.Context(), &findQueryForMi)
	if err != nil {
		reportFindDetailError("mi", err)
		return
	}
	miMap := make(map[string]reps.Mi, len(mis))
	for _, m := range mis {
		miMap[m.ID] = m
	}

	miReKyous, err := repositories.MiReKyouReps.FindMiReKyou(r.Context(), &findQueryForMi)
	if err != nil {
		reportFindDetailError("mirekyou", err)
		return
	}
	miReKyouMap := make(map[string]reps.MiReKyou, len(miReKyous))
	for _, m := range miReKyous {
		miReKyouMap[m.ID] = m
	}

	reKyous, err := repositories.ReKyouReps.FindReKyou(r.Context(), findQueryForBatch)
	if err != nil {
		reportFindDetailError("rekyou", err)
		return
	}
	reKyouMap := make(map[string]reps.ReKyou, len(reKyous))
	for _, rk := range reKyous {
		reKyouMap[rk.ID] = rk
	}

	// プラグインをrep_name別に引けるようにする。
	// プラグインKyouの本文はgkill側に保存されないため、ペイロードには
	// コンテンツHTML取得に必要なrep_name/kyou_idを載せる。
	// 申告された rep 名（get_rep_name の rep_names）でも引けるようにする。
	// zip の Git リポジトリを束ねるプラグインの Kyou はリポジトリ名を rep_name に持つ。
	pluginManifestByRepName := map[string]gkill_plugin.PluginManifest{}
	for _, pluginRep := range repositories.PluginReps {
		manifest := pluginRep.GetManifest()
		pluginManifestByRepName[manifest.RepName] = manifest
		if repNames, err := pluginRep.GetRepNames(r.Context()); err == nil {
			for _, repName := range repNames {
				pluginManifestByRepName[repName] = manifest
			}
		}
	}

	// 付随データの取得失敗を種別ごとに数える。1件でも失敗したら response.Partial を立て、
	// AIクライアントに「返した Kyou の付随データが不完全」と伝える(以前は _ で握り潰し、
	// 欠落を完全な結果に見せていた＝監査 M-05)。エラー内容は Debug ログにだけ残す(本文・IDは載せない)。
	detailFailures := map[string]int{}
	noteDetailFailure := func(kind string, err error) {
		if err == nil {
			return
		}
		detailFailures[kind]++
		slog.Log(r.Context(), gkill_log.Debug, "error at fetch attached data for mcp", "kind", kind, "error", fmt.Sprintf("%q", err))
	}

	// attached TimeIs を一括取得。削除済みの除外と「その瞬間に走っていたか」の判定は
	// どちらも get_kyous_mcp_helpers.go の関数が正本（理由と実測はそちらのコメント）。
	// 実測(2026-08-25 本番): 落とさないと付随16件のうち14件が削除済みで、最古は1年前の開始。
	// 同じ瞬間を playing_time で引くと2件しか返らない。
	var allTimeIs []reps.TimeIs
	if request.ShouldIncludeTimeIs() {
		findAllTimeIsQuery := &find.FindQuery{OnlyLatestData: true, IncludeEndTimeIs: true}
		foundTimeIs, timeisErr := repositories.TimeIsReps.FindTimeIs(r.Context(), findAllTimeIsQuery)
		noteDetailFailure("timeis", timeisErr)
		allTimeIs = livePlayingTimeIsCandidates(foundTimeIs)
	}

	// 付随 TimeIs のタグは打刻IDでメモ化する。同じ打刻が N 件の Kyou に付いても引くのは1回。
	// メモ化しないと GetTagsByTargetID の呼び出しが Σ(Kyouごとの一致件数) になる
	// (1ページ20件・1件16打刻で320回)。リクエスト単位のメモ化は ADR-0107 と同じ考え方。
	timeisTagsCache := map[string][]string{}

	// この応答に出たプラグインの rep 名。説明文を末尾に1回だけ載せるために集める。
	usedPluginRepNames := map[string]struct{}{}

	// DTO構築ループ（サイズ監視）
	maxBytes := int64(request.MaxSizeMB * 1024 * 1024)
	runningSize := int64(0)
	// 容量ヒントは candidateCount(len(batch)以下にクランプ済み)を使う。
	// go/uncontrolled-allocation-size が上限ガードを認識できるよう、割り当てを
	// candidateCount <= len(batch) が成立するブランチ内に置く(この条件は常に真)。
	// consumedCount は batch のどこまで見たか。返せた件数とは別で、
	// 直列化に失敗して落とした1件もここでは進める。カーソルはこちらを使う。
	consumedCount := 0
	resultDTOs := make([]req_res.KyouMCPDTO, 0)
	if candidateCount <= len(batch) {
		resultDTOs = make([]req_res.KyouMCPDTO, 0, candidateCount)
	}
	for i := range candidateCount {
		kyou := batch[i]

		// タグ取得
		tags, tagsErr := repositories.TagReps.GetTagsByTargetID(r.Context(), kyou.ID)
		noteDetailFailure("tags", tagsErr)
		tagStrings := make([]string, 0, len(tags))
		tagEntities := make([]req_res.AttachedEntityMCPDTO, 0, len(tags))
		for _, tag := range tags {
			tagStrings = append(tagStrings, tag.Tag)
			tagEntities = append(tagEntities, req_res.AttachedEntityMCPDTO{ID: tag.ID, Value: tag.Tag})
		}

		// テキスト取得
		texts, textsErr := repositories.TextReps.GetTextsByTargetID(r.Context(), kyou.ID)
		noteDetailFailure("texts", textsErr)
		textStrings := make([]string, 0, len(texts))
		textEntities := make([]req_res.AttachedEntityMCPDTO, 0, len(texts))
		for _, text := range texts {
			textStrings = append(textStrings, text.Text)
			textEntities = append(textEntities, req_res.AttachedEntityMCPDTO{ID: text.ID, Value: text.Text})
		}

		// 通知取得
		notifications, notificationsErr := repositories.NotificationReps.GetNotificationsByTargetID(r.Context(), kyou.ID)
		noteDetailFailure("notifications", notificationsErr)
		notificationDTOs := make([]req_res.NotificationMCPDTO, 0, len(notifications))
		for _, n := range notifications {
			notificationDTOs = append(notificationDTOs, req_res.NotificationMCPDTO{
				ID:               n.ID,
				Content:          n.Content,
				NotificationTime: localTime(n.NotificationTime),
				IsNotificated:    n.IsNotificated,
			})
		}

		// attached TimeIs 取得
		var timeisDTOs []req_res.TimeIsMCPDTO
		if request.ShouldIncludeTimeIs() && len(allTimeIs) > 0 {
			for _, ti := range allTimeIs {
				if timeIsCoversMoment(ti, kyou.RelatedTime) {
					tiTagStrings, cached := timeisTagsCache[ti.ID]
					if !cached {
						tiTags, tiTagsErr := repositories.TagReps.GetTagsByTargetID(r.Context(), ti.ID)
						noteDetailFailure("timeis_tags", tiTagsErr)
						tiTagStrings = make([]string, 0, len(tiTags))
						for _, tag := range tiTags {
							tiTagStrings = append(tiTagStrings, tag.Tag)
						}
						timeisTagsCache[ti.ID] = tiTagStrings
					}
					timeisDTOs = append(timeisDTOs, req_res.TimeIsMCPDTO{
						ID:        ti.ID,
						Title:     ti.Title,
						Tags:      tiTagStrings,
						StartTime: localTime(ti.StartTime),
						EndTime:   localTimePtr(ti.EndTime),
					})
				}
			}
		}

		// ペイロード構築。
		// DataTypeは射影ごとに枝分かれする(mi_create, timeis_start ...)ので、
		// 種別へ寄せてから分岐する。詳細は payloadKindOfDataType を参照。
		var payload any
		switch payloadKindOfDataType(kyou.DataType) {
		case "kmemo":
			if k, ok := kmemoMap[kyou.ID]; ok {
				payload = req_res.KmemoPayloadMCPDTO{
					Kind:    "kmemo",
					Content: k.Content,
				}
			}
		case "kc":
			if k, ok := kcMap[kyou.ID]; ok {
				payload = req_res.KCPayloadMCPDTO{
					Kind:     "kc",
					Title:    k.Title,
					NumValue: k.NumValue,
				}
			}
		case "timeis":
			if t, ok := timeIsMap[kyou.ID]; ok {
				payload = req_res.TimeIsPayloadMCPDTO{
					Kind:      "timeis",
					Title:     t.Title,
					StartTime: localTime(t.StartTime),
					EndTime:   localTimePtr(t.EndTime),
				}
			}
		case "nlog":
			if n, ok := nlogMap[kyou.ID]; ok {
				payload = req_res.NlogPayloadMCPDTO{
					Kind:   "nlog",
					Title:  n.Title,
					Shop:   n.Shop,
					Amount: n.Amount,
				}
			}
		case "lantana":
			if l, ok := lantanaMap[kyou.ID]; ok {
				payload = req_res.LantanaPayloadMCPDTO{
					Kind: "lantana",
					Mood: l.Mood,
				}
			}
		case "urlog":
			if u, ok := urlogMap[kyou.ID]; ok {
				payload = req_res.URLogPayloadMCPDTO{
					Kind:        "urlog",
					Title:       u.Title,
					URL:         u.URL,
					Description: u.Description,
				}
			}
		case "idf":
			if idfk, ok := idfKyouMap[kyou.ID]; ok {
				mimeType := mime.TypeByExtension(filepath.Ext(idfk.TargetFile))
				repName := kyou.RepName
				// ファイル実パスは同一マシンのクライアントにしか意味がないので、ローカルリクエストのときだけ返す
				filePath := ""
				if isLocalRequest(r) {
					filePath = idfk.ContentPath
				}
				// ファイルサイズは要求されたページ内の行だけ os.Stat で引く
				// (IDFのDBにサイズ列は無い。stat失敗はフィールド欠落のまま)
				var fileSize *int64
				if request.IncludeFileSize && idfk.ContentPath != "" {
					if stat, statErr := os.Stat(idfk.ContentPath); statErr == nil {
						size := stat.Size()
						fileSize = &size
					}
				}
				payload = req_res.IDFPayloadMCPDTO{
					Kind:     "idf",
					FileName: idfk.TargetFile,
					IsImage:  idfk.IsImage,
					IsVideo:  idfk.IsVideo,
					IsAudio:  idfk.IsAudio,
					IsZip:    idfk.IsZip,
					RepName:  repName,
					MimeType: mimeType,
					FilePath: filePath,
					FileSize: fileSize,
				}
			}
		case "git_commit_log":
			if gcl, ok := gitCommitLogMap[kyou.ID]; ok {
				payload = req_res.GitPayloadMCPDTO{
					Kind: "git_commit_log",
					// git_commit_log の Kyou ID はフル40文字のコミットハッシュそのもの
					// (git_commit_log_repository_local_dir_impl.go の kyou.ID = commit.Hash.String())
					CommitHash:    gcl.ID,
					CommitMessage: gcl.CommitMessage,
					Addition:      gcl.Addition,
					Deletion:      gcl.Deletion,
				}
			}
		case "mi":
			if m, ok := miMap[kyou.ID]; ok {
				payload = req_res.MiPayloadMCPDTO{
					Kind:              "mi",
					Title:             m.Title,
					IsChecked:         m.IsChecked,
					BoardName:         m.BoardName,
					CreateTime:        localTime(m.CreateTime),
					LimitTime:         localTimePtr(m.LimitTime),
					EstimateStartTime: localTimePtr(m.EstimateStartTime),
					EstimateEndTime:   localTimePtr(m.EstimateEndTime),
				}
			}
		case "mirekyou":
			if m, ok := miReKyouMap[kyou.ID]; ok {
				payload = req_res.MiReKyouPayloadMCPDTO{
					Kind:              "mirekyou",
					TargetID:          m.TargetID,
					IsChecked:         m.IsChecked,
					BoardName:         m.BoardName,
					CreateTime:        localTime(m.CreateTime),
					LimitTime:         localTimePtr(m.LimitTime),
					EstimateStartTime: localTimePtr(m.EstimateStartTime),
					EstimateEndTime:   localTimePtr(m.EstimateEndTime),
				}
			}
		case "rekyou":
			if rk, ok := reKyouMap[kyou.ID]; ok {
				payload = req_res.ReKyouPayloadMCPDTO{
					Kind:     "rekyou",
					TargetID: rk.TargetID,
				}
			}
		default:
			// 既存のdata_typeに該当しないKyouはプラグイン由来。
			// data_typeはプラグインが自由に決めるので、rep_nameで引き当てる。
			if manifest, ok := pluginManifestByRepName[kyou.RepName]; ok {
				payload = req_res.PluginPayloadMCPDTO{
					Kind:       "plugin",
					DataType:   kyou.DataType,
					RepName:    kyou.RepName,
					KyouID:     kyou.ID,
					PluginName: manifest.Name,
				}
				// 説明文は応答トップレベルへ rep_name ごと1回だけ（DTOのコメント参照）。
				if _, seen := usedPluginRepNames[kyou.RepName]; !seen {
					usedPluginRepNames[kyou.RepName] = struct{}{}
				}
			}
		}

		dto := req_res.KyouMCPDTO{
			ID:            kyou.ID,
			DataType:      kyou.DataType,
			RepName:       kyou.RepName,
			CreateApp:     kyou.CreateApp,
			UpdateApp:     kyou.UpdateApp,
			RelatedTime:   kyou.RelatedTime.In(time.Local),
			IsDeleted:     kyou.IsDeleted,
			UpdateTime:    kyou.UpdateTime.In(time.Local),
			Tags:          tagStrings,
			Texts:         textStrings,
			Notifications: notificationDTOs,
			TimeIs:        timeisDTOs,
			TagEntities:   tagEntities,
			TextEntities:  textEntities,
			Payload:       payload,
		}

		dtoJSON, marshalErr := json.Marshal(dto)
		if marshalErr != nil {
			// この1件は返せないが、消費位置は進める。進めないと次ページの
			// カーソルが返却済みより手前を指し、返した分をもう一度返す。
			// 返却数と全件数の差はこの1件ぶんずれるので、黙って落とさず警告する。
			consumedCount = i + 1
			response.Warnings = append(response.Warnings, fmt.Sprintf(
				"record %s could not be serialized and was skipped; total_count and the number of returned records differ by it", kyou.ID))
			continue
		}

		// ★v2ではMaxSizeMBは厳密な上限（複合カーソルが同一時刻のかたまりの途中からでも
		//   再開できるため、旧v1の「かたまりを跨ぐまで入れ続ける」例外は不要になった）。
		//   唯一の例外はページ先頭の1件が単独で上限を超えるとき: 返さないと
		//   0件+has_more=true の永久ループになるため、その1件だけ返して警告する。
		if len(resultDTOs) > 0 && runningSize+int64(len(dtoJSON)) > maxBytes {
			break
		}
		if len(resultDTOs) == 0 && int64(len(dtoJSON)) > maxBytes {
			response.Warnings = append(response.Warnings, fmt.Sprintf(
				"single record (%d bytes) exceeds max_size_mb (%d bytes); returned anyway to keep pagination progressing", len(dtoJSON), maxBytes))
		}
		runningSize += int64(len(dtoJSON))
		consumedCount = i + 1
		resultDTOs = append(resultDTOs, dto)
	}

	returnedCount := len(resultDTOs)
	// 残件数もカーソルも「どこまで見たか」で決める。返せた件数で決めると、
	// 直列化に失敗して落とした1件のぶんだけカーソルが後戻りする。
	remainingCount := len(batch) - consumedCount
	hasMore := remainingCount > 0
	nextCursor := ""
	if hasMore && consumedCount > 0 {
		// 複合カーソル {RFC3339Nano}::{ID}。
		// 時刻がRFC3339Nanoなのは、秒へ切り捨てると同じ秒の内側が漏れるため。
		// IDを併記することで、同一時刻のかたまりの途中でページを割っても
		// 次ページが正確な位置から再開できる（これがLimit厳密化の前提。ADR-0604）。
		last := batch[consumedCount-1]
		nextCursor = encodeMCPCursor(last.RelatedTime, last.ID)
	}

	response.Kyous = resultDTOs
	// TotalCount は cursor 無しの応答のみ（カーソル押し下げ後のハンドラは全件数を知らない）。
	// 全応答に RemainingCount。旧v1のTotalCountはカーソルの有無で意味が変わっていた（外部監査 S3）。
	if !hasCursor {
		response.TotalCount = &totalCount
	}
	response.ReturnedCount = returnedCount
	response.RemainingCount = remainingCount
	response.HasMore = hasMore
	response.NextCursor = nextCursor
	// この応答に出たプラグインの説明を rep 名ごとに1回だけ載せる。
	// payload へ焼き込むと130〜150字が Kyou の件数ぶん並ぶ（req_res 側のコメント参照）。
	for repName := range usedPluginRepNames {
		manifest, ok := pluginManifestByRepName[repName]
		if !ok {
			continue
		}
		response.Plugins = append(response.Plugins, req_res.PluginDescriptionMCPDTO{
			RepName:     repName,
			PluginName:  manifest.Name,
			Description: manifest.Description,
		})
	}
	slices.SortFunc(response.Plugins, func(a, b req_res.PluginDescriptionMCPDTO) int {
		return strings.Compare(a.RepName, b.RepName)
	})
	// 付随データの取得に1件でも失敗していたら、返した結果が不完全であることを明示する。
	if len(detailFailures) > 0 {
		response.Partial = true
		kinds := make([]string, 0, len(detailFailures))
		for kind := range detailFailures {
			kinds = append(kinds, kind)
		}
		slices.Sort(kinds)
		for _, kind := range kinds {
			response.Warnings = append(response.Warnings, fmt.Sprintf("failed to fetch %s for %d record(s); attached data is incomplete", kind, detailFailures[kind]))
		}
	}
	appendGetKyousMCPSuccess(response, request.LocaleName)
}

// appendGetKyousMCPSuccess は成功メッセージを積む（count_only / group_by の早期returnと共有）。
func appendGetKyousMCPSuccess(response *req_res.GetKyousMCPResponse, localeName string) {
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKyousMCPSuccessMessage,
		Message:     api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KYOUS_MESSAGE"}),
	})
}

// payloadKindOfDataType は射影ごとに枝分かれしたDataTypeをペイロード種別へ寄せます。
//
// Mi / MiReKyou / TimeIs のDataTypeはリポジトリのSQLが射影名を焼き込むため
// (mi_create / mi_check / mi_limit / mi_start / mi_end、mirekyou_*、
// timeis_start / timeis_end)、素の "mi" / "timeis" とは一致しません。
// さらに FindFilter.overrideKyous がMi検索時にMiSortTypeへ合わせて付け替えます。
// 完全一致のswitchで書くと全射影がdefaultへ落ち、payloadごと消えます。
//
// mirekyou_ を mi_ より先に判定すること（接頭辞判定の順序の罠）。
// 単体取得経路は素の "mi" / "timeis" を使うので、既知の射影に当たらない値は
// そのまま返して呼び出し側のcaseに任せます。
func payloadKindOfDataType(dataType string) string {
	switch {
	case strings.HasPrefix(dataType, "mirekyou_"):
		return "mirekyou"
	case strings.HasPrefix(dataType, "mi_"):
		return "mi"
	case strings.HasPrefix(dataType, "timeis_"):
		return "timeis"
	}
	return dataType
}
