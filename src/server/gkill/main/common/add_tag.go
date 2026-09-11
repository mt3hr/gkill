package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
)

// addTagAppName はタグのCREATE_APP/UPDATE_APPへ刻む名前。
//
// サブコマンドは2026-09に auto_tag から add_tag へ改名したが、**この値は変えていない**。
// 付与元をあとから見分けるための値なので、変えると過去に付けたぶんと出所が食い違う。
// さらに addTagIDNamespace の材料でもあるので、変えるとタグIDまで食い違い、
// 再実行が AlreadyExistTagError で止まらずに全件付け直しになる。
const addTagAppName = "gkill_auto_tag"

// addTagLocaleName はAPIへ渡すロケール。エラーメッセージの取得にしか使わない。
const addTagLocaleName = "ja"

// addTagIDNamespace は自動付与したタグのIDを決めるための名前空間。
//
// 同じ(対象ID, タグ名)には常に同じIDを振る。これが冪等性の要で、
// 「付いているか」の判定を取りこぼしても、サーバ側が同じIDのタグを
// AlreadyExistTagErrorで弾くので二重登録にならない。
// 論理削除されたタグも同じIDで弾かれるため、消したタグが復活することもない。
//
// 文字列を変えると過去に付与したぶんとIDが食い違い、全件が付け直しになる。
var addTagIDNamespace = uuid.NewSHA1(uuid.NameSpaceOID, []byte("github.com/mt3hr/gkill/"+addTagAppName))

var (
	addTagRuleArgs      []string
	addTagRulesFileArgs []string
	addTagDryRun        bool
)

// addTagRuleJSON はルール1件のJSON表現。`{"tag": "...", "query": {...}}`。
type addTagRuleJSON struct {
	Tag   string           `json:"tag"`
	Query *addTagQueryJSON `json:"query"`
}

// addTagQueryJSON はクライアント(src/client)の FindKyouQuery の JSON をそのまま受ける型。
//
// find.FindQuery を無タグで埋め込むことで、サーバが受ける全キーが既知のフィールドになる。
// それに加えてクライアント専用のキー(サーバへは送られない、サイドバーの状態など)を並べる。
// 復号は DisallowUnknownFields で行うので、クライアント側にキーが増えたらここにも足すこと
// (add_tag_test.go の TestAddTagQueryJSONAcceptsEveryClientFindKyouQueryKey が検出する)。
type addTagQueryJSON struct {
	find.FindQuery
	QueryID                    string   `json:"query_id"`
	Keywords                   string   `json:"keywords"`
	TimeIsKeywords             string   `json:"timeis_keywords"`
	DevicesInSidebar           []string `json:"devices_in_sidebar"`
	RepTypesInSidebar          []string `json:"rep_types_in_sidebar"`
	IsEnableMapCircleInSidebar bool     `json:"is_enable_map_circle_in_sidebar"`
	IsFocusKyouInListView      bool     `json:"is_focus_kyou_in_list_view"`
}

// addTagRule は検証済みのルール。「Query に一致する Kyou のうち Tag が付いていないものへ Tag を付ける」。
type addTagRule struct {
	Tag   string
	Query find.FindQuery
	// RepTypesInSidebar はクライアントの「記録種別」(dvnf名の先頭要素)。
	// nil なら展開しない。非nilなら rep 名一覧から Query.Reps を作り直す。
	RepTypesInSidebar []string
	// DevicesInSidebar は RepTypesInSidebar の展開で端末を絞る集合。nil なら全端末。
	DevicesInSidebar []string
	// Source はエラーや進捗の表示に使う出どころ("--rule #1" / "rules.json #2")。
	Source string
}

// addTagTarget はタグを付ける対象のKyouと、付けるタグ名。
type addTagTarget struct {
	Kyou reps.Kyou
	Tags []string
}

// AddTagCmd は検索条件(FindKyouQuery)に一致する Kyou へタグを付ける。
//
// ルールは `{"tag": "<タグ名>", "query": {<FindKyouQuery JSON>}}` で、
// --rule に直接書くか、--rules_file にその配列を書いたファイルを渡す。
// query はクライアントの検索条件 JSON そのもので、rep_types_in_sidebar(記録種別)を
// 書くと rep 名一覧から reps を展開してから検索する。
//
// 判定も付与も稼働中のgkill_serverのAPI越しに行う。
// 認証はissueLocalSessionで発行する対象ユーザの短命セッション
// (APIはセッションのユーザとして動くので、管理者セッションでは対象ユーザのrepを見られない)。
//
// すでに同じタグが付いているKyouには何もしないので、何度実行してもよい。
var AddTagCmd = &cobra.Command{
	Use:   "add_tag",
	Short: `add_tag 'user_id' --rules_file <path> | --rule '<json>'`,
	Long: `add_tag 'user_id'... --rules_file <path> | --rule '<json>' [--dry_run]

検索条件に一致する Kyou のうち、指定のタグが付いていないものへタグを付ける。
ルールは {"tag": "<タグ名>", "query": {<FindKyouQuery JSON>}}。
--rules_file にはその配列(単一オブジェクトでも可)を書いたファイルを渡す。

例:
  [{"tag": "autolog_screenshot", "query": {"rep_types_in_sidebar": ["AutoScreenshot"]}}]

query の rep_types_in_sidebar は画面の「記録種別」(rep 名 <種別>_<端末>_<日付> の先頭要素)で、
稼働中サーバの rep 名一覧から reps を展開してから検索する。
devices_in_sidebar を省略すると全端末が対象になる。
絞り込みが1つも無い query、綴りの違うキー、常に0件になる指定(tags: [] など)は受け付けない。`,
	Args:          cobra.ArbitraryArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 指定の誤りは stderr にも出す。RunE のエラーは main の gkill_log.Fatal がログファイルへ
		// 書くだけで端末には出ないため、それだけだと usage と exit 1 しか見えず、
		// ルールのどこが悪いのかが利用者に伝わらない。
		if len(args) == 0 {
			// 変数の未定義などで user_id が1つも届かないケース。
			// usage を見せるだけで nil を返すと exit 0 になり、呼び出し側のスクリプトが気付けない。
			err := errors.New("user_id を1つ以上指定してください")
			fmt.Fprintf(os.Stderr, "add_tag: %v\n", err)
			cmd.Usage()
			return err
		}
		ctx := cmd.Context()

		// ルールはサーバに触る前に全部読んで検証する(途中のルールの書式誤りで半端に付けない)。
		rules, err := loadAddTagRules(addTagRuleArgs, addTagRulesFileArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "add_tag: %v\n", err)
			cmd.Usage()
			return err
		}

		endpoint, err := ResolveLocalServerEndpoint(ctx)
		if err != nil {
			return fmt.Errorf("error at resolve local server endpoint: %w", err)
		}
		configDBRootDir := os.ExpandEnv(gkill_options.ConfigDir)

		// 1ユーザが失敗しても残りは続け、最後にまとめて返す。
		// os.Exit(1)で抜けると defer(cleanupSession) を飛ばして短命セッションを消し損ねるので使わない。
		var errs []error
		for _, userID := range args {
			if err := addTagForUser(ctx, endpoint, configDBRootDir, userID, rules); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	},
}

func init() {
	// StringArrayVar を使う。StringSliceVar だと JSON の中のカンマで分割される。
	AddTagCmd.Flags().StringArrayVar(&addTagRuleArgs, "rule", nil, `'{"tag":"<tag>","query":{<FindKyouQuery JSON>}}' (repeatable)`)
	AddTagCmd.Flags().StringArrayVar(&addTagRulesFileArgs, "rules_file", nil, "path to a JSON file holding a rule or an array of rules (repeatable)")
	AddTagCmd.Flags().BoolVar(&addTagDryRun, "dry_run", false, "print what would be added without calling add_tag")
}

// loadAddTagRules は --rules_file と --rule を読み、検証済みのルールをその順に返す。
func loadAddTagRules(ruleArgs []string, fileArgs []string) ([]addTagRule, error) {
	rules := []addTagRule{}
	for _, path := range fileArgs {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("error at read rules file %s: %w", path, err)
		}
		parsed, err := parseAddTagRulesJSON(raw, path)
		if err != nil {
			return nil, err
		}
		rules = append(rules, parsed...)
	}
	for i, arg := range ruleArgs {
		parsed, err := parseAddTagRulesJSON([]byte(arg), fmt.Sprintf("--rule #%d", i+1))
		if err != nil {
			return nil, err
		}
		rules = append(rules, parsed...)
	}
	if len(rules) == 0 {
		return nil, errors.New("--rule か --rules_file でルールを1つ以上指定してください")
	}
	return rules, nil
}

// parseAddTagRulesJSON はルールのJSON(配列または単一オブジェクト)を復号して検証する。
// source はエラー表示に使う出どころ。
func parseAddTagRulesJSON(raw []byte, source string) ([]addTagRule, error) {
	wires, err := decodeAddTagRulesStrict(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	rules := make([]addTagRule, 0, len(wires))
	for i, wire := range wires {
		// 出どころは複数件のときだけ何件目かを添える("--rule #1 #1" のような二重表示を避ける)
		ruleSource := source
		if len(wires) > 1 {
			ruleSource = fmt.Sprintf("%s #%d", source, i+1)
		}
		rule, err := buildAddTagRule(wire, ruleSource)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// decodeAddTagRulesStrict はルールのJSONを厳格に復号する。
//
//   - 先頭のBOMは剥がす(Windows のエディタが付けがち。Go のデコーダはBOMを読めない)
//   - 旧形式(use_* フラグ)の検索条件は find.MigrateLegacyFindQueryJSON で現行形式へ直す
//   - 未知のキーはエラーにする。"rep" のような綴り誤りを黙って落とすと、
//     そのフィルタが「未使用」になって全件が対象になる
//   - JSON 値の後ろに余分な内容があればエラー(`{...}{...}` を先頭の1件として通さない)
func decodeAddTagRulesStrict(raw []byte) ([]addTagRuleJSON, error) {
	raw = bytes.TrimPrefix(raw, []byte("\xEF\xBB\xBF"))
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, errors.New("ルールのJSONが空です")
	}
	// MigrateLegacyFindQueryJSON は先頭の1値しか見ないので、余分な内容の検査はその前に行う
	if err := ensureSingleJSONValue(trimmed); err != nil {
		return nil, err
	}
	migrated, _, err := find.MigrateLegacyFindQueryJSON(trimmed)
	if err != nil {
		return nil, fmt.Errorf("ルールのJSONを読めません: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(migrated))
	decoder.DisallowUnknownFields()
	var wires []addTagRuleJSON
	switch trimmed[0] {
	case '[':
		if err := decoder.Decode(&wires); err != nil {
			return nil, fmt.Errorf("ルールのJSONを読めません(キーの綴りも確認してください): %w", err)
		}
	case '{':
		var wire addTagRuleJSON
		if err := decoder.Decode(&wire); err != nil {
			return nil, fmt.Errorf("ルールのJSONを読めません(キーの綴りも確認してください): %w", err)
		}
		wires = []addTagRuleJSON{wire}
	default:
		return nil, errors.New(`ルールのJSONは {"tag": "...", "query": {...}} かその配列で書いてください`)
	}
	if len(wires) == 0 {
		return nil, errors.New("ルールが0件です")
	}
	return wires, nil
}

// ensureSingleJSONValue は raw が JSON 値ちょうど1つであることを確かめる。
func ensureSingleJSONValue(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var probe json.RawMessage
	if err := decoder.Decode(&probe); err != nil {
		return fmt.Errorf("ルールのJSONを読めません: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("ルールのJSONの後ろに余分な内容があります")
	}
	return nil
}

// buildAddTagRule は復号したルールを検証して addTagRule にする。
//
// ここで弾くのは「エラーにならず、意図と違う範囲に静かに当たる」書き方。
// 全件に当たる側(絞り込み無し)も、0件になる側(空配列)も、黙って通すと気付けない。
func buildAddTagRule(wire addTagRuleJSON, source string) (addTagRule, error) {
	fail := func(format string, args ...any) (addTagRule, error) {
		return addTagRule{}, fmt.Errorf("%s: %s", source, fmt.Sprintf(format, args...))
	}

	tag := strings.TrimSpace(wire.Tag)
	if tag == "" {
		return fail("tag が空です")
	}
	q := wire.Query
	if q == nil {
		return fail("query がありません")
	}

	// keywords はクライアントの入力欄の生文字列で、送信直前に words / not_words へ解析される。
	// CLI では解析しないので、非空のまま受けると黙って無視されて広く当たる
	if q.Keywords != "" || q.TimeIsKeywords != "" {
		return fail("keywords / timeis_keywords は使えません。words / not_words (timeis_words / timeis_not_words) を書いてください")
	}
	if q.DevicesInSidebar != nil && q.RepTypesInSidebar == nil {
		return fail("devices_in_sidebar は rep_types_in_sidebar と一緒に指定してください")
	}
	// update_cache は検索のたびにサーバのキャッシュを丸ごと作り直す副作用
	if q.UpdateCache {
		return fail("update_cache は指定できません(検索のたびにキャッシュを作り直します)。キャッシュは先に update_cache サブコマンドで更新してください")
	}

	// 常に0件になる指定(非nilの空配列)。フィルタを使わないなら null かキー省略
	for _, spec := range []struct {
		key   string
		empty bool
	}{
		{"tags", q.Tags != nil && len(q.Tags) == 0},
		{"rep_types", q.RepTypes != nil && len(q.RepTypes) == 0},
		{"ids", q.IDs != nil && len(q.IDs) == 0},
		{"period_of_time_week_of_days", q.PeriodOfTimeWeekOfDays != nil && len(q.PeriodOfTimeWeekOfDays) == 0},
		{"reps", q.Reps != nil && len(q.Reps) == 0 && q.RepTypesInSidebar == nil},
	} {
		if spec.empty {
			return fail("%s: [] は常に0件です(絞らないなら null かキー省略)", spec.key)
		}
	}

	// 揃っていないと黙って無視される組み合わせ
	if (q.MapRadius != nil || q.MapLatitude != nil || q.MapLongitude != nil) && !q.HasMapFilter() {
		return fail("map_latitude / map_longitude / map_radius は3つ揃えて指定してください")
	}
	if q.TimeIsTags != nil && !q.HasTimeIsFilter() {
		return fail("timeis_tags は timeis_words (空配列でよい) と一緒に指定してください")
	}

	if q.RepTypesInSidebar == nil && isFilterlessFindQuery(&q.FindQuery) {
		return fail("検索条件が空です(全件が対象になります)。絞り込みを1つ以上書いてください")
	}

	return addTagRule{
		Tag:               tag,
		Query:             q.FindQuery,
		RepTypesInSidebar: q.RepTypesInSidebar,
		DevicesInSidebar:  q.DevicesInSidebar,
		Source:            source,
	}, nil
}

// isFilterlessFindQuery は絞り込みが1つも無い(=全件に当たる)検索条件かを返す。
//
// 数えるのは対象を狭めるフィールドだけ。*_and / include_* / mi_sort_type / only_latest_data /
// include_deleted_data は修飾子、mi_board_name / mi_check_state / timeis_tags は
// 別のフィルタの配下でしか効かないので数えない。
func isFilterlessFindQuery(q *find.FindQuery) bool {
	return q.RepTypes == nil && q.IDs == nil && q.Reps == nil && q.Tags == nil &&
		len(q.HideTags) == 0 &&
		!q.HasWordFilter() && !q.HasTimeIsFilter() && !q.HasCalendarFilter() &&
		!q.HasMapFilter() && !q.HasPeriodOfTimeFilter() &&
		q.PlayingTime == nil && q.UpdateTime == nil &&
		!q.IsImageOnly && !q.ForMi
}

// splitRepNameLikeClient はクライアントの rep_to_struct
// (src/client/classes/api/find_query/find-kyou-query.ts)と同じ規則で rep 名を種別と端末に分ける。
// "_" で3要素に分かれれば (種別, 端末)、それ以外は (rep名そのもの, "なし")。
func splitRepNameLikeClient(repName string) (repType string, device string) {
	parts := strings.Split(repName, "_")
	if len(parts) != 3 {
		return repName, "なし"
	}
	return parts[0], parts[1]
}

// expandSidebarReps は rep 名一覧から、種別が repTypes に含まれ、
// かつ端末が devices に含まれる(devices が nil なら全端末)rep 名を重複なく昇順で返す。
// クライアントの apply_rep_summary_sets_to_detaul と同じ選び方。当たらなければ空(非nil)。
func expandSidebarReps(allRepNames []string, repTypes []string, devices []string) []string {
	typeSet := make(map[string]struct{}, len(repTypes))
	for _, repType := range repTypes {
		typeSet[repType] = struct{}{}
	}
	var deviceSet map[string]struct{}
	if devices != nil {
		deviceSet = make(map[string]struct{}, len(devices))
		for _, device := range devices {
			deviceSet[device] = struct{}{}
		}
	}

	selected := map[string]struct{}{}
	for _, repName := range allRepNames {
		repType, device := splitRepNameLikeClient(repName)
		if _, ok := typeSet[repType]; !ok {
			continue
		}
		if deviceSet != nil {
			if _, ok := deviceSet[device]; !ok {
				continue
			}
		}
		selected[repName] = struct{}{}
	}

	result := make([]string, 0, len(selected))
	for repName := range selected {
		result = append(result, repName)
	}
	slices.Sort(result)
	return result
}

// effectiveQuery はサーバへ送る検索条件を返す。
// RepTypesInSidebar が非nilなら Reps を rep 名一覧からの展開で上書きする(クライアントと同じ)。
// レシーバの Query は書き換えない。
func (r *addTagRule) effectiveQuery(allRepNames []string) *find.FindQuery {
	query := r.Query
	if r.RepTypesInSidebar != nil {
		query.Reps = expandSidebarReps(allRepNames, r.RepTypesInSidebar, r.DevicesInSidebar)
	}
	return &query
}

// formatEffectiveQuery は検索条件を確認用の1行JSONにする。
// null / false / "" のフィールドは落として、指定したものだけが見えるようにする
// (空配列と 0 は意味を持つので残す)。
func formatEffectiveQuery(query *find.FindQuery) string {
	full, err := json.Marshal(query)
	if err != nil {
		return fmt.Sprintf("(marshal error: %v)", err)
	}
	fields := map[string]any{}
	if err := json.Unmarshal(full, &fields); err != nil {
		return string(full)
	}
	for key, value := range fields {
		switch v := value.(type) {
		case nil:
			delete(fields, key)
		case bool:
			if !v {
				delete(fields, key)
			}
		case string:
			if v == "" {
				delete(fields, key)
			}
		}
	}
	compact, err := json.Marshal(fields)
	if err != nil {
		return string(full)
	}
	return string(compact)
}

// addTagForUser は1ユーザぶんのタグ付与を行う。
func addTagForUser(ctx context.Context, endpoint *LocalServerEndpoint, configDBRootDir string, userID string, rules []addTagRule) error {
	sessionID, refreshSession, cleanupSession, err := issueLocalSession(ctx, configDBRootDir, endpoint.Device, userID)
	if err != nil {
		return fmt.Errorf("error at issue local session user id = %s: %w", userID, err)
	}
	defer cleanupSession()

	client := &addTagAPIClient{Endpoint: endpoint, SessionID: sessionID}
	targets := map[string]*addTagTarget{}

	// rep 名一覧は展開が要るルールが1つでもあるときだけ、1回取る
	var allRepNames []string
	if slices.ContainsFunc(rules, func(rule addTagRule) bool { return rule.RepTypesInSidebar != nil }) {
		allRepNames, err = client.GetAllRepNames(ctx)
		if err != nil {
			return fmt.Errorf("error at get all rep names user id = %s: %w", userID, err)
		}
	}

	for i := range rules {
		rule := &rules[i]
		// 収集は1ルールにつき検索2回で、条件によっては長い。セッションのTTL(5分)を跨がないよう、
		// ルールごとに延長する。失敗しても続ける(次の区切りで挽回できる)
		if i > 0 && refreshSession != nil {
			if err := refreshSession(); err != nil {
				slog.Log(ctx, gkill_log.Warn, "error at refresh add_tag session ttl", "user_id", fmt.Sprintf("%q", userID), "error", fmt.Sprintf("%q", err))
			}
		}

		query := rule.effectiveQuery(allRepNames)
		fmt.Printf("%s: %s tag = %s query = %s\n", userID, rule.Source, rule.Tag, formatEffectiveQuery(query))
		if query.Reps != nil && len(query.Reps) == 0 {
			// 展開で1つも当たらなかった。サーバへ投げても0件なので投げない
			fmt.Printf("%s: %s no rep matched: rep_types_in_sidebar = %s\n", userID, rule.Source, strings.Join(rule.RepTypesInSidebar, ", "))
			continue
		}
		if err := collectByQuery(ctx, client, userID, rule, query, targets); err != nil {
			return err
		}
	}

	if len(targets) == 0 {
		fmt.Printf("%s: no target\n", userID)
		return nil
	}
	return addTags(ctx, client, userID, targets, refreshSession)
}

// addTagRefreshInterval は、この件数タグを付けるごとにセッションのTTLを延長する間隔。
// 進捗印字と同じ区切り。大量付与でセッションが期限切れになるのを防ぐ。
const addTagRefreshInterval = 500

// shouldRefreshAddTagSession は、これまでに付けた件数addedがrefresh間隔の区切りかを返す。
// added==0(まだ1件も付けていない)では延長しない。
func shouldRefreshAddTagSession(added int) bool {
	return added > 0 && added%addTagRefreshInterval == 0
}

// collectByQuery は1ルールぶんの対象を集める。
// 「条件そのまま」と「そのタグが付いているものだけ」の2回の検索の差分が、まだ付いていないKyou。
func collectByQuery(ctx context.Context, client *addTagAPIClient, userID string, rule *addTagRule, query *find.FindQuery, targets map[string]*addTagTarget) error {
	allKyous, err := client.GetKyous(ctx, query)
	if err != nil {
		return fmt.Errorf("error at find kyous %s user id = %s: %w", rule.Source, userID, err)
	}
	taggedIDs, err := client.FindTaggedKyouIDs(ctx, query, rule.Tag)
	if err != nil {
		return fmt.Errorf("error at find tagged kyous %s user id = %s: %w", rule.Source, userID, err)
	}

	added := 0
	for _, kyou := range allKyous {
		if _, exist := taggedIDs[kyou.ID]; exist {
			continue
		}
		putAddTagTarget(targets, kyou, rule.Tag)
		added++
	}

	repsSummary := ""
	if query.Reps != nil {
		repsSummary = fmt.Sprintf(" reps = %d (%s)", len(query.Reps), strings.Join(query.Reps, ", "))
	}
	fmt.Printf("%s: %s tag = %s%s kyous = %d tagged = %d target = %d\n",
		userID, rule.Source, rule.Tag, repsSummary, len(allKyous), len(taggedIDs), added)
	return nil
}

// putAddTagTarget は付与予定へ1件積む。同じタグを二度積まない。
func putAddTagTarget(targets map[string]*addTagTarget, kyou reps.Kyou, tagName string) {
	target, exist := targets[kyou.ID]
	if !exist {
		target = &addTagTarget{Kyou: kyou}
		targets[kyou.ID] = target
	}
	if slices.Contains(target.Tags, tagName) {
		return
	}
	target.Tags = append(target.Tags, tagName)
}

// addTagID は(対象ID, タグ名)から決まるタグのIDを返す。
func addTagID(targetID string, tagName string) string {
	return uuid.NewSHA1(addTagIDNamespace, []byte(targetID+"\x00"+tagName)).String()
}

// addTags は集めた対象へタグを付ける。
// refresh は500件ごとの進捗印字に相乗りしてセッションのTTLを延ばす(長時間実行での期限切れ防止)。
func addTags(ctx context.Context, client *addTagAPIClient, userID string, targets map[string]*addTagTarget, refresh func() error) error {
	runAt := time.Now()

	targetIDs := make([]string, 0, len(targets))
	for targetID := range targets {
		targetIDs = append(targetIDs, targetID)
	}
	// 途中で止めて再開したときに同じ順で進むよう、並びを決めておく
	slices.Sort(targetIDs)

	added, alreadyExist := 0, 0
	for _, targetID := range targetIDs {
		target := targets[targetID]
		for _, tagName := range target.Tags {
			tag := reps.Tag{
				IsDeleted: false,
				ID:        addTagID(targetID, tagName),
				TargetID:  targetID,
				Tag:       tagName,
				// タグの関連時刻は対象のKyouの時刻に合わせる
				RelatedTime:  target.Kyou.RelatedTime,
				CreateTime:   runAt,
				CreateApp:    addTagAppName,
				CreateDevice: client.Endpoint.Device,
				CreateUser:   userID,
				UpdateTime:   runAt,
				UpdateApp:    addTagAppName,
				UpdateDevice: client.Endpoint.Device,
				UpdateUser:   userID,
			}

			if addTagDryRun {
				fmt.Printf("(dry run) add tag: target = %s tag = %s rep = %s\n", targetID, tagName, target.Kyou.RepName)
				added++
				continue
			}

			exist, err := client.AddTag(ctx, tag)
			if err != nil {
				return fmt.Errorf("error at add tag target = %s tag = %s user id = %s: %w", targetID, tagName, userID, err)
			}
			if exist {
				// 同じIDのタグが既にある。消されたタグを付け直さないための経路でもあるので、失敗ではない
				alreadyExist++
				continue
			}
			added++
			if shouldRefreshAddTagSession(added) {
				fmt.Printf("%s: added %d tags\n", userID, added)
				// セッションのTTLを延長する。失敗しても付与は続ける(次の区切りで挽回できる)。
				if refresh != nil {
					if err := refresh(); err != nil {
						slog.Log(ctx, gkill_log.Warn, "error at refresh add_tag session ttl", "user_id", fmt.Sprintf("%q", userID), "error", fmt.Sprintf("%q", err))
					}
				}
			}
		}
	}

	fmt.Printf("%s: added = %d already_exist = %d elapsed = %s\n", userID, added, alreadyExist, time.Since(runAt).String())
	return nil
}

// addTagAPIClient は稼働中のgkill_serverのAPIを叩く。
type addTagAPIClient struct {
	Endpoint  *LocalServerEndpoint
	SessionID string
	// MaxResponseBodyBytes は応答本文を読む上限。0 なら defaultMaxResponseBodyBytes。
	// テストが小さな値で上限超過の壊れ方を確かめるために持たせている。
	MaxResponseBodyBytes int64
}

// defaultMaxResponseBodyBytes は応答本文を読む上限の既定値。
// /api/get_kyous は検索条件に一致した Kyou を全件返すので、任意の検索条件を受ける以上
// 実用的な上限は張れない(30万件で百数十MB)。暴走を止める意味で1GiBにしてある。
// io.LimitReader は上限超過をエラーにせず黙って打ち切るので、超えると途中で切れたJSONの
// デコード失敗として現れる。
const defaultMaxResponseBodyBytes = 1 << 30

// post はJSONをPOSTして、レスポンスをresponseへ書き込む。
func (c *addTagAPIClient) post(ctx context.Context, path string, requestBody any, response any) error {
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error at marshal request for %s: %w", path, err)
	}

	address := c.Endpoint.BaseURL + path
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("error at new request for %s: %w", address, err)
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := c.Endpoint.Client.Do(request)
	if err != nil {
		return fmt.Errorf("error at post %s: %w", address, err)
	}
	defer resp.Body.Close()

	// **先にデコードしてから、ステータスと応答の両方を見る。**
	// gkillは2026-08から異常時に4xx/5xxを返すが、エラーの中身(error_code)は
	// 今までどおり本文のerrors配列にしか入っていない。ステータスで打ち切ると
	// 「HTTP 401」しか分からず、セッション切れなのか権限不足なのか判別できなくなる。
	// 逆にHTTP 200でもerrorsに中身が入ることがあるので、両方を見る必要がある
	// (update_cache側のcommon.goと同じ形)。
	limit := c.MaxResponseBodyBytes
	if limit <= 0 {
		limit = defaultMaxResponseBodyBytes
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, limit))
	if readErr != nil {
		return fmt.Errorf("error at read response of %s: status = %d: %w", address, resp.StatusCode, readErr)
	}
	if decodeErr := json.Unmarshal(body, response); decodeErr != nil {
		// 本文がJSONですらない場合(TLSのサーバへ平文で繋いだ等)は、
		// ステータスと本文の断片を添えて返す。
		snippet := body
		if len(snippet) > 1024 {
			snippet = snippet[:1024]
		}
		return fmt.Errorf("error at decode response of %s: status = %d body = %q: %w", address, resp.StatusCode, string(snippet), decodeErr)
	}
	if resp.StatusCode != http.StatusOK {
		// 非2xxでも、本文のerrorsに中身があるならここでは打ち切らない。
		// エラーの意味はerror_codeにしか入っておらず、その判定は呼び出し側にしかできない。
		// 特にAddTagはERR000056(既存ID)を「既に付いている」として飲む必要があり、
		// 2026-08からERR000056はHTTP 409で届くため、ここで打ち切ると
		// 冪等なはずの再実行が最初の既存タグで失敗するようになってしまう。
		// それ以外の呼び出し側もaddTagResponseErrorでerror_message込みのエラーにする
		// (本文のerrorsを優先する判断はMCPのgkill-client.mjsと同じ)。
		// 本文にエラーの中身が無いときだけ、ステータスを唯一の手掛かりとして返す。
		probe := struct {
			Errors []*message.GkillError `json:"errors"`
		}{}
		if json.Unmarshal(body, &probe) == nil && addTagResponseError(path, probe.Errors) != nil {
			return nil
		}
		return fmt.Errorf("error at post %s: status = %d", address, resp.StatusCode)
	}
	return nil
}

// GetAllRepNames はrep名の一覧を取る。
func (c *addTagAPIClient) GetAllRepNames(ctx context.Context) ([]string, error) {
	response := &req_res.GetAllRepNamesResponse{}
	err := c.post(ctx, "/api/get_all_rep_names", &req_res.GetAllRepNamesRequest{
		SessionID:  c.SessionID,
		LocaleName: addTagLocaleName,
	}, response)
	if err != nil {
		return nil, err
	}
	if err := addTagResponseError("/api/get_all_rep_names", response.Errors); err != nil {
		return nil, err
	}
	return response.RepNames, nil
}

// addTagGetKyousWarningCodes は /api/get_kyous の messages のうち印字する警告のコード。
// 成功メッセージ(GetKyousSuccessMessage「検索完了」)は毎回載るので出さない。
var addTagGetKyousWarningCodes = []string{
	message.FindKyousPluginWarningMessage,
	message.FindKyousRepLoadWarningMessage,
}

// GetKyous は検索条件に一致するKyouを取る。
// 応答の messages のうち警告(読み込めなかった rep・失敗したプラグイン)は成功扱いのまま印字する。
// 黙って捨てると、その rep の記録が対象から消えていることに気付けない。
func (c *addTagAPIClient) GetKyous(ctx context.Context, query *find.FindQuery) ([]reps.Kyou, error) {
	response := &req_res.GetKyousResponse{}
	err := c.post(ctx, "/api/get_kyous", &req_res.GetKyousRequest{
		SessionID:  c.SessionID,
		Query:      query,
		LocaleName: addTagLocaleName,
	}, response)
	if err != nil {
		return nil, err
	}
	if err := addTagResponseError("/api/get_kyous", response.Errors); err != nil {
		return nil, err
	}
	for _, gkillMessage := range response.Messages {
		if gkillMessage == nil || !slices.Contains(addTagGetKyousWarningCodes, gkillMessage.MessageCode) {
			continue
		}
		fmt.Printf("warn: /api/get_kyous %s %s\n", gkillMessage.MessageCode, gkillMessage.Message)
	}
	return response.Kyous, nil
}

// buildTaggedQuery は「tagNameが付いているものだけ」に絞った検索条件を組み立てる。
//
// TagsAndを立てる。現在のfind_filterはOR/ANDどちらの分岐もタグ名を完全一致
// (大文字小文字無視)で照合するためどちらでも同じ結果になるが、
// 単一タグの「付いているものだけ」という意図はANDの方が直接表現になるためこちらを使う。
// (かつてはOR側が部分一致で"gkill"に"gkill_autolog"まで誤ヒットしたと記録されていたが、
// 現行コードでは両分岐とも完全一致であることを確認済み)
//
// 渡されたqueryのタグ以外の条件(reps / 期間 / hide_tags など)はそのまま残るので、
// 「条件そのまま」の結果との差分がそのまま「まだ付いていないもの」になる。
// 渡されたquery自体は書き換えない。
func (c *addTagAPIClient) buildTaggedQuery(query *find.FindQuery, tagName string) *find.FindQuery {
	taggedQuery := *query
	taggedQuery.Tags = []string{tagName}
	taggedQuery.TagsAnd = true
	return &taggedQuery
}

// FindTaggedKyouIDs は、渡した条件のうちtagNameが付いているKyouのIDを返す。
func (c *addTagAPIClient) FindTaggedKyouIDs(ctx context.Context, query *find.FindQuery, tagName string) (map[string]struct{}, error) {
	kyous, err := c.GetKyous(ctx, c.buildTaggedQuery(query, tagName))
	if err != nil {
		return nil, err
	}
	taggedIDs := make(map[string]struct{}, len(kyous))
	for _, kyou := range kyous {
		taggedIDs[kyou.ID] = struct{}{}
	}
	return taggedIDs, nil
}

// AddTag はタグを1件付ける。
// 同じIDのタグが既にある場合はtrueを返す(失敗ではない)。
func (c *addTagAPIClient) AddTag(ctx context.Context, tag reps.Tag) (alreadyExist bool, err error) {
	response := &req_res.AddTagResponse{}
	err = c.post(ctx, "/api/add_tag", &req_res.AddTagRequest{
		SessionID:  c.SessionID,
		Tag:        tag,
		LocaleName: addTagLocaleName,
	}, response)
	if err != nil {
		return false, err
	}
	for _, gkillError := range response.Errors {
		if gkillError != nil && gkillError.ErrorCode == message.AlreadyExistTagError {
			return true, nil
		}
	}
	if err := addTagResponseError("/api/add_tag", response.Errors); err != nil {
		return false, err
	}
	return false, nil
}

// addTagResponseError は応答のerrorsを1つのerrorにまとめる。
// gkillはHTTP 200でもerrorsに中身を入れることがあるので、必ず見る。
func addTagResponseError(path string, gkillErrors []*message.GkillError) error {
	if len(gkillErrors) == 0 {
		return nil
	}
	errorMessages := make([]string, 0, len(gkillErrors))
	for _, gkillError := range gkillErrors {
		if gkillError == nil {
			continue
		}
		errorMessages = append(errorMessages, gkillError.ErrorCode+" "+gkillError.ErrorMessage)
	}
	if len(errorMessages) == 0 {
		return nil
	}
	return fmt.Errorf("error at %s: %s", path, strings.Join(errorMessages, " / "))
}
