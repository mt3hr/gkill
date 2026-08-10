package sqlite3impl

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

func TestEscapeSQLite_SingleQuote(t *testing.T) {
	result := EscapeSQLite("it's")
	if result != "it''s" {
		t.Errorf("EscapeSQLite(\"it's\") = %q, want %q", result, "it''s")
	}
}

func TestEscapeSQLite_NoQuotes(t *testing.T) {
	result := EscapeSQLite("hello world")
	if result != "hello world" {
		t.Errorf("EscapeSQLite(\"hello world\") = %q, want %q", result, "hello world")
	}
}

func TestEscapeSQLite_MultipleQuotes(t *testing.T) {
	result := EscapeSQLite("it's a 'test'")
	if result != "it''s a ''test''" {
		t.Errorf("EscapeSQLite(\"it's a 'test'\") = %q, want %q", result, "it''s a ''test''")
	}
}

func TestEscapeSQLite_EmptyString(t *testing.T) {
	result := EscapeSQLite("")
	if result != "" {
		t.Errorf("EscapeSQLite(\"\") = %q, want %q", result, "")
	}
}

func TestEscapeSQLite_JapaneseText(t *testing.T) {
	result := EscapeSQLite("テスト'データ")
	if result != "テスト''データ" {
		t.Errorf("EscapeSQLite(\"テスト'データ\") = %q, want %q", result, "テスト''データ")
	}
}

func TestQuoteIdent_Simple(t *testing.T) {
	result := QuoteIdent("column_name")
	expected := `"column_name"`
	if result != expected {
		t.Errorf("QuoteIdent(\"column_name\") = %q, want %q", result, expected)
	}
}

func TestQuoteIdent_WithDoubleQuotes(t *testing.T) {
	result := QuoteIdent(`col"name`)
	expected := `"col""name"`
	if result != expected {
		t.Errorf("QuoteIdent(\"col\\\"name\") = %q, want %q", result, expected)
	}
}

func TestGenerateNewID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for range 100 {
		id := GenerateNewID()
		if id == "" {
			t.Fatal("GenerateNewID returned empty string")
		}
		if ids[id] {
			t.Fatalf("GenerateNewID produced duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestTimeLayout_IsValid(t *testing.T) {
	if TimeLayout == "" {
		t.Fatal("TimeLayout is empty")
	}
	// Should be Go RFC3339-like format
	expected := "2006-01-02T15:04:05-07:00"
	if TimeLayout != expected {
		t.Errorf("TimeLayout = %q, want %q", TimeLayout, expected)
	}
}

func TestGenerateFindSQLCommon_EmptyQuery(t *testing.T) {
	query := &find.FindQuery{}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty query should produce a tautology " 0 = 0 "
	if !strings.Contains(sql, "0 = 0") {
		t.Errorf("expected tautology '0 = 0' in sql, got %q", sql)
	}
	if len(queryArgs) != 0 {
		t.Errorf("expected 0 queryArgs for empty query, got %d", len(queryArgs))
	}
}

func TestGenerateFindSQLCommon_IDsSpecified(t *testing.T) {
	query := &find.FindQuery{
		IDs: []string{"id-1", "id-2", "id-3"},
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "ID IN (") {
		t.Errorf("expected 'ID IN (' in sql, got %q", sql)
	}
	// Should have 3 query args for the 3 IDs
	if len(queryArgs) != 3 {
		t.Errorf("expected 3 queryArgs, got %d", len(queryArgs))
	}
	for i, expected := range []string{"id-1", "id-2", "id-3"} {
		if queryArgs[i] != expected {
			t.Errorf("queryArgs[%d] = %v, want %v", i, queryArgs[i], expected)
		}
	}
}

func TestGenerateFindSQLCommon_IDsEmptySpecified(t *testing.T) {
	query := &find.FindQuery{
		IDs: []string{},
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 非nilの空スライス = 「明示的に0件指定」なので矛盾式になる
	if !strings.Contains(sql, "0 = 1") {
		t.Errorf("expected '0 = 1' for empty IDs, got %q", sql)
	}
	// 「0 = 1」に続けて区切りなしで「0 = 0」を出すとSQL構文エラーになる。
	// 空のrep（最新版アドレスが1件も無いrep）の検索で必ず踏むので固定しておく。
	if strings.Contains(sql, "0 = 0") {
		t.Errorf("'0 = 1' の後に '0 = 0' を続けてはいけない (構文エラーになる), got %q", sql)
	}
	if err := assertValidWhereClause(t, sql, queryArgs); err != nil {
		t.Errorf("生成されたWHERE句がSQLiteで実行できない: %v (sql=%q)", err, sql)
	}
}

// assertValidWhereClause は生成されたWHERE句が実際にSQLiteでパースできることを確認します。
// 文字列一致だけだと今回のような構文エラーを取りこぼすため。
func assertValidWhereClause(t *testing.T, whereSQL string, args []any) error {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE MY_TABLE (ID, TITLE, RELATED_TIME, UPDATE_TIME)`); err != nil {
		return err
	}
	rows, err := db.Query(`SELECT ID FROM MY_TABLE AS T WHERE `+whereSQL, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

func TestGenerateFindSQLCommon_WordsSpecified(t *testing.T) {
	query := &find.FindQuery{
		Words:    []string{"hello"},
		WordsAnd: true,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "LIKE") {
		t.Errorf("expected LIKE in sql for word search, got %q", sql)
	}
	// Should have args for word search (TITLE LIKE and ID LIKE)
	if len(queryArgs) < 2 {
		t.Errorf("expected at least 2 queryArgs for word search, got %d", len(queryArgs))
	}
	// First arg should be the word wrapped with %
	if queryArgs[0] != "%hello%" {
		t.Errorf("queryArgs[0] = %v, want %%hello%%", queryArgs[0])
	}
}

func TestGenerateFindSQLCommon_WordsOrSpecified(t *testing.T) {
	query := &find.FindQuery{
		Words:    []string{"foo", "bar"},
		WordsAnd: false,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "LIKE") {
		t.Errorf("expected LIKE in sql for OR word search, got %q", sql)
	}
	// Each word produces 2 args (column LIKE + ID LIKE), 2 words = 4 args
	if len(queryArgs) != 4 {
		t.Errorf("expected 4 queryArgs for 2-word OR search, got %d", len(queryArgs))
	}
}

func TestGenerateFindSQLCommon_CalendarSpecified(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.Local)
	query := &find.FindQuery{
		CalendarStartDate: &start,
		CalendarEndDate:   &end,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 時刻の比較は unixepoch() で行う。
	// datetime(列,'localtime') は非決定的で式インデックスに使えず、
	// 列に関数を適用しているため索引がまったく効かなかった。
	if !strings.Contains(sql, "unixepoch(RELATED_TIME)") {
		t.Errorf("expected 'unixepoch(RELATED_TIME)' in sql for calendar search, got %q", sql)
	}
	if strings.Contains(sql, "datetime(RELATED_TIME") {
		t.Errorf("時刻列に datetime() を適用すると索引が効かない, got %q", sql)
	}
	if !strings.Contains(sql, ">=") {
		t.Errorf("expected '>=' in sql for calendar start date, got %q", sql)
	}
	if !strings.Contains(sql, "<=") {
		t.Errorf("expected '<=' in sql for calendar end date, got %q", sql)
	}
	if len(queryArgs) != 2 {
		t.Errorf("expected 2 queryArgs for calendar range, got %d", len(queryArgs))
	}
}

func TestGenerateFindSQLCommon_OnlyLatestData(t *testing.T) {
	query := &find.FindQuery{}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		true, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "UPDATE_TIME = ( SELECT UPDATE_TIME FROM") || !strings.Contains(sql, "ORDER BY datetime(INNER_TABLE.UPDATE_TIME) DESC LIMIT 1") {
		t.Errorf("expected latest data subquery with datetime() in sql, got %q", sql)
	}
}

func TestGenerateFindSQLCommon_AppendOrderBy(t *testing.T) {
	query := &find.FindQuery{}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		true, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 文字列の時刻列は unixepoch() で並べる。
	// オフセットが混在していると文字列順が時系列にならないうえ、
	// WHERE側と式を揃えないと式インデックスが並び替えに使われない。
	if !strings.Contains(sql, "ORDER BY unixepoch(RELATED_TIME) DESC") {
		t.Errorf("expected ORDER BY unixepoch(RELATED_TIME) DESC in sql, got %q", sql)
	}
}

// キャッシュ側の _UNIX 列は既に整数なので、そのまま並べる。
func TestGenerateFindSQLCommon_AppendOrderByUnixColumn(t *testing.T) {
	query := &find.FindQuery{}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME_UNIX",
		[]string{"TITLE"}, true, false,
		true, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "ORDER BY RELATED_TIME_UNIX DESC") {
		t.Errorf("expected ORDER BY RELATED_TIME_UNIX DESC in sql, got %q", sql)
	}
	if strings.Contains(sql, "unixepoch(RELATED_TIME_UNIX)") {
		t.Errorf("整数列に unixepoch() を適用してはいけない, got %q", sql)
	}
}

func TestGenerateFindSQLCommon_IgnoreCase(t *testing.T) {
	query := &find.FindQuery{
		Words:    []string{"Test"},
		WordsAnd: true,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, true, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "LOWER") {
		t.Errorf("expected LOWER in sql for case-insensitive search, got %q", sql)
	}
}

func TestGenerateFindSQLCommon_NotWords(t *testing.T) {
	query := &find.FindQuery{
		NotWords: []string{"exclude"},
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "NOT LIKE") {
		t.Errorf("expected 'NOT LIKE' in sql for not-words, got %q", sql)
	}
}

// matchedIDsOfTwoColumnTable は (ID, TITLE, SHOP) の表にrowsを入れ、
// 生成されたWHERE句に一致するIDを返します。
// 文字列一致では「列どうしをANDでつないでいる」ような論理の誤りを検出できないため、
// 実際にSQLiteに投げて結果集合で確かめます。
func matchedIDsOfTwoColumnTable(t *testing.T, whereSQL string, args []any, rows [][3]string) []string {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("error at open memory db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE MY_TABLE (ID, TITLE, SHOP, RELATED_TIME, UPDATE_TIME)`); err != nil {
		t.Fatalf("error at create table: %v", err)
	}
	for _, row := range rows {
		if _, err := db.Exec(`INSERT INTO MY_TABLE (ID, TITLE, SHOP) VALUES (?, ?, ?)`, row[0], row[1], row[2]); err != nil {
			t.Fatalf("error at insert row %v: %v", row, err)
		}
	}

	selected, err := db.Query(`SELECT ID FROM MY_TABLE AS T WHERE `+whereSQL, args...)
	if err != nil {
		t.Fatalf("error at select with generated where clause %q: %v", whereSQL, err)
	}
	defer func() { _ = selected.Close() }()

	matchedIDs := []string{}
	for selected.Next() {
		id := ""
		if err := selected.Scan(&id); err != nil {
			t.Fatalf("error at scan: %v", err)
		}
		matchedIDs = append(matchedIDs, id)
	}
	if err := selected.Err(); err != nil {
		t.Fatalf("error at iterate rows: %v", err)
	}
	return matchedIDs
}

// and検索は「語ごとにAND・列どうしはOR」でなければなりません。
// 外側を列にしてANDで連結していたころは「全列に含む」の意味になっており、
// URLog(URL/TITLE/DESCRIPTION)やNlog(TITLE/SHOP)のような複数列repで、
// 片方の列にしか無い語が検索結果から落ちていました。
func TestGenerateFindSQLCommon_WordsAndMultipleColumns(t *testing.T) {
	query := &find.FindQuery{
		Words:    []string{"github"},
		WordsAnd: true,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE", "SHOP"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	matchedIDs := matchedIDsOfTwoColumnTable(t, sql, queryArgs, [][3]string{
		{"only-title", "github page", "somewhere"},
		{"only-shop", "some page", "github shop"},
		{"both", "github page", "github shop"},
		{"neither", "some page", "somewhere"},
	})

	want := map[string]bool{"only-title": true, "only-shop": true, "both": true}
	if len(matchedIDs) != len(want) {
		t.Fatalf("片方の列にしか語が無い行も一致しなければならない: got %v, want %v", matchedIDs, want)
	}
	for _, id := range matchedIDs {
		if !want[id] {
			t.Errorf("語をどの列にも含まない行が一致した: %q (matched=%v)", id, matchedIDs)
		}
	}
}

// 語がすべて揃っていなければ一致しないこと（and検索であること）も確かめます。
// 列どうしをORにした結果、語どうしまでORになっていないことの確認。
func TestGenerateFindSQLCommon_WordsAndMultipleColumnsRequiresAllWords(t *testing.T) {
	query := &find.FindQuery{
		Words:    []string{"github", "gkill"},
		WordsAnd: true,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE", "SHOP"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	matchedIDs := matchedIDsOfTwoColumnTable(t, sql, queryArgs, [][3]string{
		{"split-over-columns", "github page", "gkill shop"},
		{"only-one-word", "github page", "somewhere"},
	})

	if len(matchedIDs) != 1 || matchedIDs[0] != "split-over-columns" {
		t.Errorf("全ての語が（列をまたいでよいので）揃っている行だけが一致すべき: got %v", matchedIDs)
	}
}

// ignoreFindWord は「SQLでは絞らずGo側の判定に任せる」の意味です。
// IDFRepのようにrep内相対パスや .md/.txt の本文まで見るリポジトリで、
// SQLが先にファイル名だけで絞ってしまうとGo側の判定に到達できなくなります。
func TestGenerateFindSQLCommon_IgnoreFindWordSkipsWordSQL(t *testing.T) {
	query := &find.FindQuery{
		Words:    []string{"hello"},
		NotWords: []string{"exclude"},
		WordsAnd: true,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, true,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(sql, "LIKE") {
		t.Errorf("ignoreFindWordのときはキーワードのSQLを組み立ててはいけない, got %q", sql)
	}
	if len(queryArgs) != 0 {
		t.Errorf("ignoreFindWordのときはキーワードのバインド値を積んではいけない, got %v", queryArgs)
	}
	if err := assertValidWhereClause(t, sql, queryArgs); err != nil {
		t.Errorf("生成されたWHERE句がSQLiteで実行できない: %v (sql=%q)", err, sql)
	}
}

// 検索対象列を持たないrep(Lantana等)は、他のrepと同じくID列だけをキーワードの対象にします。
// ignoreFindWord の値によらないこと。
// 以前は無条件で '1 = 0' を出力しており、ID検索が効かないうえ、
// 除外語(NotWords)だけの検索でも全件が消えていました。
func TestGenerateFindSQLCommon_NoFindWordTargetColumnsMatchesIDOnly(t *testing.T) {
	for _, ignoreFindWord := range []bool{true, false} {
		query := &find.FindQuery{
			Words:    []string{"hello"},
			WordsAnd: true,
		}
		whereCounter := 0
		queryArgs := []any{}

		sql, err := GenerateFindSQLCommon(
			query, "MY_TABLE", "T", &whereCounter,
			false, "RELATED_TIME",
			[]string{}, true, ignoreFindWord,
			false, false, &queryArgs,
		)
		if err != nil {
			t.Fatalf("unexpected error (ignoreFindWord=%v): %v", ignoreFindWord, err)
		}
		if !strings.Contains(sql, "(ID) LIKE") {
			t.Errorf("検索対象列が無いときはID列だけを対象にするはず (ignoreFindWord=%v), got %q", ignoreFindWord, sql)
		}
		if len(queryArgs) != 1 {
			t.Errorf("バインド値はIDの1個のはず (ignoreFindWord=%v), got %v", ignoreFindWord, queryArgs)
		}
		if err := assertValidWhereClause(t, sql, queryArgs); err != nil {
			t.Errorf("生成されたWHERE句がSQLiteで実行できない (ignoreFindWord=%v): %v (sql=%q)", ignoreFindWord, err, sql)
		}
	}
}

// 検索対象列を持たないrepは、除外語(NotWords)だけの検索では全件が残ります。
// 本文が無いので除外語に該当しえないため。以前は '1 = 0' で全件が消えていました。
func TestGenerateFindSQLCommon_NoFindWordTargetColumnsNotWordsOnlyPasses(t *testing.T) {
	query := &find.FindQuery{
		Words:    []string{},
		NotWords: []string{"exclude"},
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(sql, "1 = 0") {
		t.Errorf("除外語だけの検索で全件を消してはいけない, got %q", sql)
	}
	if !strings.Contains(sql, "(ID) NOT LIKE") {
		t.Errorf("除外語はID列を対象にするはず, got %q", sql)
	}
	if err := assertValidWhereClause(t, sql, queryArgs); err != nil {
		t.Errorf("生成されたWHERE句がSQLiteで実行できない: %v (sql=%q)", err, sql)
	}
}

// LIKEパターンの % _ \ はエスケープされ、リテラルとして扱われます。
// 以前は未エスケープで「100%」が前方一致に化け、除外語「%」で全件が消えていました。
func TestGenerateFindSQLCommon_LikePatternMetacharactersAreEscaped(t *testing.T) {
	query := &find.FindQuery{
		Words:    []string{"100%", "snake_case"},
		WordsAnd: false,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, true, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "ESCAPE") {
		t.Errorf("LIKEにはESCAPE句が付くはず, got %q", sql)
	}
	foundEscapedPercent := false
	foundEscapedUnderscore := false
	for _, arg := range queryArgs {
		s, ok := arg.(string)
		if !ok {
			continue
		}
		if strings.Contains(s, `\%`) {
			foundEscapedPercent = true
		}
		if strings.Contains(s, `\_`) {
			foundEscapedUnderscore = true
		}
	}
	if !foundEscapedPercent {
		t.Errorf("%% はエスケープされるはず, got %v", queryArgs)
	}
	if !foundEscapedUnderscore {
		t.Errorf("_ はエスケープされるはず, got %v", queryArgs)
	}
	if err := assertValidWhereClause(t, sql, queryArgs); err != nil {
		t.Errorf("生成されたWHERE句がSQLiteで実行できない: %v (sql=%q)", err, sql)
	}
}

// update_time=null（未使用）なら panic せず、Calendar条件へ倒れること
func TestGenerateFindSQLCommon_UpdateTimeNilFallsBackToCalendar(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
	query := &find.FindQuery{
		UpdateTime:        nil,
		CalendarStartDate: &start,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, true, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "RELATED_TIME") {
		t.Errorf("UpdateTime未指定ならCalendar条件が適用されるはず, got %q", sql)
	}
	if err := assertValidWhereClause(t, sql, queryArgs); err != nil {
		t.Errorf("生成されたWHERE句がSQLiteで実行できない: %v (sql=%q)", err, sql)
	}
}

// weekdayIDs は曜日番号（日曜=0）から行IDへの対応。
var weekdayIDs = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

// matchedWeekdayIDsOfTable は曜日ごとに1行ずつ（日〜土の7行）持つ表を作り、
// 生成されたWHERE句に一致した行のID集合を返します。
// 曜日の判定は strftime('%w', datetime(列, 'localtime')) で行われるので、
// 文字列一致ではなく実際にSQLiteへ投げて結果集合で確かめます。
func matchedWeekdayIDsOfTable(t *testing.T, whereSQL string, args []any) map[string]bool {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("error at open memory db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE MY_TABLE (ID, TITLE, RELATED_TIME, UPDATE_TIME)`); err != nil {
		t.Fatalf("error at create table: %v", err)
	}

	// 連続する7日を入れるので、開始日の曜日によらず日〜土が1行ずつ揃う
	baseTime := time.Date(2026, 8, 3, 12, 0, 0, 0, time.Local)
	for i := range 7 {
		rowTime := baseTime.AddDate(0, 0, i)
		id := weekdayIDs[int(rowTime.Weekday())]
		if _, err := db.Exec(`INSERT INTO MY_TABLE (ID, TITLE, RELATED_TIME) VALUES (?, ?, ?)`, id, "title", rowTime.Format(TimeLayout)); err != nil {
			t.Fatalf("error at insert row %s: %v", id, err)
		}
	}

	selected, err := db.Query(`SELECT ID FROM MY_TABLE AS T WHERE `+whereSQL, args...)
	if err != nil {
		t.Fatalf("error at select with generated where clause %q: %v", whereSQL, err)
	}
	defer func() { _ = selected.Close() }()

	matchedIDs := map[string]bool{}
	for selected.Next() {
		id := ""
		if err := selected.Scan(&id); err != nil {
			t.Fatalf("error at scan: %v", err)
		}
		matchedIDs[id] = true
	}
	if err := selected.Err(); err != nil {
		t.Fatalf("error at iterate rows: %v", err)
	}
	return matchedIDs
}

// 時間帯フィルタの曜日指定は nil=曜日で絞らない / 非nilの空スライス=0件 / 全7曜日=絞らない。
//
// nil を先に外さないと len==0 の分岐（ 0 = 1 ）へ落ちて全件が消えます。
// 「曜日を指定していない」と「曜日を1つもチェックしていない」は
// Use* フラグ廃止後は値のnil判定でしか区別できないので、4通りとも固定します。
//
// 全ケースで時刻の範囲（0:00:00〜23:59:59＝全行が通る）も一緒に指定しています。
// 曜日がnilかつ時刻の範囲も未指定だと HasPeriodOfTimeFilter() が偽になって
// 時間帯フィルタ全体が素通しになり、nil先行ガードまで到達しないためです。
func TestGenerateFindSQLCommon_PeriodOfTimeWeekOfDays(t *testing.T) {
	allWeekdays := map[string]bool{}
	for _, id := range weekdayIDs {
		allWeekdays[id] = true
	}

	// 時刻部分（時:分:秒）だけが使われるので日付は何でもよい
	periodStartTimeSecond := time.Date(2026, 8, 3, 0, 0, 0, 0, time.Local).Unix()
	periodEndTimeSecond := time.Date(2026, 8, 3, 23, 59, 59, 0, time.Local).Unix()

	cases := []struct {
		name       string
		weekOfDays []find.WeekOfDays
		want       map[string]bool
	}{
		{
			name:       "nilは曜日で絞らない",
			weekOfDays: nil,
			want:       allWeekdays,
		},
		{
			name:       "非nilの空スライスは0件",
			weekOfDays: []find.WeekOfDays{},
			want:       map[string]bool{},
		},
		{
			name:       "全7曜日は絞らないのと同じ",
			weekOfDays: []find.WeekOfDays{find.SunDay, find.MonDay, find.TuesDay, find.WednesDay, find.ThursDay, find.FriDay, find.SaturDay},
			want:       allWeekdays,
		},
		{
			name:       "一部の曜日だけ",
			weekOfDays: []find.WeekOfDays{find.MonDay, find.WednesDay, find.SaturDay},
			want:       map[string]bool{"mon": true, "wed": true, "sat": true},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			query := &find.FindQuery{
				PeriodOfTimeStartTimeSecond: &periodStartTimeSecond,
				PeriodOfTimeEndTimeSecond:   &periodEndTimeSecond,
				PeriodOfTimeWeekOfDays:      c.weekOfDays,
			}
			whereCounter := 0
			queryArgs := []any{}

			sql, err := GenerateFindSQLCommon(
				query, "MY_TABLE", "T", &whereCounter,
				false, "RELATED_TIME",
				[]string{"TITLE"}, true, false,
				false, false, &queryArgs,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			matchedIDs := matchedWeekdayIDsOfTable(t, sql, queryArgs)
			if len(matchedIDs) != len(c.want) {
				t.Fatalf("一致した曜日 = %v, want %v (sql=%q)", matchedIDs, c.want, sql)
			}
			for id := range c.want {
				if !matchedIDs[id] {
					t.Errorf("%s が一致していない: matched=%v (sql=%q)", id, matchedIDs, sql)
				}
			}
		})
	}
}

// EscapeLikePattern の変換規則
func TestEscapeLikePattern(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{`100%`, `100\%`},
		{`snake_case`, `snake\_case`},
		{`back\slash`, `back\\slash`},
		{`plain`, `plain`},
		{`%_\`, `\%\_` + `\\`},
	}
	for _, c := range cases {
		if got := EscapeLikePattern(c.input); got != c.want {
			t.Errorf("EscapeLikePattern(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
