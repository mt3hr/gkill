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

// 検索対象列を持たない呼び出し（ID指定で1件を引く内部クエリ）は、他のrepと同じくID列だけを
// キーワードの対象にします（前方一致）。ignoreFindWord の値によらないこと。
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
		if len(queryArgs) != 1 || queryArgs[0] != "hello%" {
			t.Errorf("バインド値はIDの前方一致パターン1個のはず (ignoreFindWord=%v), got %v", ignoreFindWord, queryArgs)
		}
		if err := assertValidWhereClause(t, sql, queryArgs); err != nil {
			t.Errorf("生成されたWHERE句がSQLiteで実行できない (ignoreFindWord=%v): %v (sql=%q)", ignoreFindWord, err, sql)
		}
	}
}

// 検索対象列を持たない呼び出しは、除外語(NotWords)だけの検索では全件が残ります。
// 本文が無いので除外語に該当しえないため（除外語は ID を見ない）。以前は '1 = 0' で全件が消えていました。
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
	if strings.Contains(sql, "NOT LIKE") {
		t.Errorf("除外語はID列を見ないので条件を出さないはず, got %q", sql)
	}
	if len(queryArgs) != 0 {
		t.Errorf("除外語だけならバインド値を積まないはず, got %v", queryArgs)
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

// 時間帯フィルタの狭い窓（09:00〜10:00）を、秒オブデイ表現とepoch表現の両方 ×
// 文字列列(RELATED_TIME)と数値列(RELATED_TIME_UNIX)の両方で、実SQLiteに投げて固定する。
//
// バインド値は find.SecondOfDayToHHMMSS の "HH:MM:SS" 文字列で、列側の
// strftime('%H:%M:%S', ...) と文字列比較される。以前はバインド側が
// 「epoch秒をそのままdatetime()に食わせる」形で、秒オブデイを渡すと
// 1970-01-01の時刻として+9時間ずれていた（指摘で発覚）。
// 解釈の正本: find.NormalizeSecondOfDay / documents/adr/0108-period-of-time-second-of-day.md
func TestGenerateFindSQLCommon_PeriodOfTimeNarrowWindow(t *testing.T) {
	day := time.Date(2026, 8, 19, 0, 0, 0, 0, time.Local)
	rows := map[string]time.Time{
		"before":    day.Add(8*time.Hour + 59*time.Minute + 59*time.Second),
		"at-start":  day.Add(9 * time.Hour),
		"inside":    day.Add(9*time.Hour + 30*time.Minute),
		"at-end":    day.Add(10 * time.Hour),
		"after-end": day.Add(10*time.Hour + 1*time.Second),
	}
	want := map[string]bool{"at-start": true, "inside": true, "at-end": true}

	secOfDayStart := int64(9 * 3600)
	secOfDayEnd := int64(10 * 3600)
	epochStart := time.Date(2026, 1, 1, 9, 0, 0, 0, time.Local).Unix()
	epochEnd := time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local).Unix()

	matchedIDs := func(t *testing.T, columnName string, whereSQL string, args []any) map[string]bool {
		t.Helper()
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatalf("error at open memory db: %v", err)
		}
		defer func() { _ = db.Close() }()
		if _, err := db.Exec(`CREATE TABLE MY_TABLE (ID, TITLE, RELATED_TIME, RELATED_TIME_UNIX, UPDATE_TIME)`); err != nil {
			t.Fatalf("error at create table: %v", err)
		}
		for id, rowTime := range rows {
			if _, err := db.Exec(`INSERT INTO MY_TABLE (ID, TITLE, RELATED_TIME, RELATED_TIME_UNIX) VALUES (?, ?, ?, ?)`,
				id, "title", rowTime.Format(TimeLayout), rowTime.Unix()); err != nil {
				t.Fatalf("error at insert row %s: %v", id, err)
			}
		}
		selected, err := db.Query(`SELECT ID FROM MY_TABLE AS T WHERE `+whereSQL, args...)
		if err != nil {
			t.Fatalf("error at select with generated where clause %q: %v", whereSQL, err)
		}
		defer func() { _ = selected.Close() }()
		got := map[string]bool{}
		for selected.Next() {
			id := ""
			if err := selected.Scan(&id); err != nil {
				t.Fatalf("error at scan: %v", err)
			}
			got[id] = true
		}
		if err := selected.Err(); err != nil {
			t.Fatalf("error at iterate rows: %v", err)
		}
		return got
	}

	for _, expr := range []struct {
		name       string
		columnName string
	}{
		{name: "文字列列", columnName: "RELATED_TIME"},
		{name: "数値列(_UNIX)", columnName: "RELATED_TIME_UNIX"},
	} {
		for _, c := range []struct {
			name       string
			start, end int64
		}{
			{name: "秒オブデイ表現(MCP契約)", start: secOfDayStart, end: secOfDayEnd},
			{name: "epoch表現(Web契約)", start: epochStart, end: epochEnd},
		} {
			t.Run(expr.name+"/"+c.name, func(t *testing.T) {
				query := &find.FindQuery{
					PeriodOfTimeStartTimeSecond: &c.start,
					PeriodOfTimeEndTimeSecond:   &c.end,
				}
				whereCounter := 0
				queryArgs := []any{}
				whereSQL, err := GenerateFindSQLCommon(
					query, "MY_TABLE", "T", &whereCounter,
					false, expr.columnName,
					[]string{"TITLE"}, true, false,
					false, false, &queryArgs,
				)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				got := matchedIDs(t, expr.columnName, whereSQL, queryArgs)
				if len(got) != len(want) {
					t.Fatalf("一致した行 = %v, want %v (sql=%q args=%v)", got, want, whereSQL, queryArgs)
				}
				for id := range want {
					if !got[id] {
						t.Errorf("%s が一致していない: matched=%v (sql=%q)", id, got, whereSQL)
					}
				}
			})
		}
	}
}

// 夜跨ぎ窓（23:00〜01:00）のSQL経路。start > end で OR 判定に切り替わる分岐を両表現で固定する。
func TestGenerateFindSQLCommon_PeriodOfTimeOvernightWindow(t *testing.T) {
	day := time.Date(2026, 8, 19, 0, 0, 0, 0, time.Local)
	rows := map[string]time.Time{
		"evening-out": day.Add(22*time.Hour + 59*time.Minute + 59*time.Second),
		"at-start":    day.Add(23 * time.Hour),
		"early":       day.Add(24*time.Hour + 30*time.Minute),
		"at-end":      day.Add(25 * time.Hour),
		"after-end":   day.Add(25*time.Hour + 1*time.Second),
		"midday-out":  day.Add(12 * time.Hour),
	}
	want := map[string]bool{"at-start": true, "early": true, "at-end": true}

	secOfDayStart := int64(23 * 3600)
	secOfDayEnd := int64(1 * 3600)
	epochStart := time.Date(2026, 1, 1, 23, 0, 0, 0, time.Local).Unix()
	epochEnd := time.Date(2026, 1, 1, 1, 0, 0, 0, time.Local).Unix()

	for _, c := range []struct {
		name       string
		start, end int64
	}{
		{name: "秒オブデイ表現(MCP契約)", start: secOfDayStart, end: secOfDayEnd},
		{name: "epoch表現(Web契約)", start: epochStart, end: epochEnd},
	} {
		t.Run(c.name, func(t *testing.T) {
			query := &find.FindQuery{
				PeriodOfTimeStartTimeSecond: &c.start,
				PeriodOfTimeEndTimeSecond:   &c.end,
			}
			whereCounter := 0
			queryArgs := []any{}
			whereSQL, err := GenerateFindSQLCommon(
				query, "MY_TABLE", "T", &whereCounter,
				false, "RELATED_TIME",
				[]string{"TITLE"}, true, false,
				false, false, &queryArgs,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatalf("error at open memory db: %v", err)
			}
			defer func() { _ = db.Close() }()
			if _, err := db.Exec(`CREATE TABLE MY_TABLE (ID, TITLE, RELATED_TIME, UPDATE_TIME)`); err != nil {
				t.Fatalf("error at create table: %v", err)
			}
			for id, rowTime := range rows {
				if _, err := db.Exec(`INSERT INTO MY_TABLE (ID, TITLE, RELATED_TIME) VALUES (?, ?, ?)`, id, "title", rowTime.Format(TimeLayout)); err != nil {
					t.Fatalf("error at insert row %s: %v", id, err)
				}
			}
			selected, err := db.Query(`SELECT ID FROM MY_TABLE AS T WHERE `+whereSQL, queryArgs...)
			if err != nil {
				t.Fatalf("error at select with generated where clause %q: %v", whereSQL, err)
			}
			defer func() { _ = selected.Close() }()
			got := map[string]bool{}
			for selected.Next() {
				id := ""
				if err := selected.Scan(&id); err != nil {
					t.Fatalf("error at scan: %v", err)
				}
				got[id] = true
			}
			if err := selected.Err(); err != nil {
				t.Fatalf("error at iterate rows: %v", err)
			}
			if len(got) != len(want) {
				t.Fatalf("一致した行 = %v, want %v (sql=%q args=%v)", got, want, whereSQL, queryArgs)
			}
			for id := range want {
				if !got[id] {
					t.Errorf("%s が一致していない: matched=%v (sql=%q)", id, got, whereSQL)
				}
			}
		})
	}
}

// 肯定語の ID 照合は前方一致だけ。
// 部分一致だったころは `1` / `a` のような hex だけの短い語が UUID に偶然含まれ、
// 本文と無関係な記録が「ランダムに」出ていた。UUID 丸ごとの貼り付けと git の短縮ハッシュは前方一致で引ける。
func TestGenerateFindSQLCommon_IDMatchesByPrefixOnly(t *testing.T) {
	query := &find.FindQuery{
		Words:    []string{"abc"},
		WordsAnd: false,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE", "SHOP"}, true, false,
		false, true, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	matchedIDs := matchedIDsOfTwoColumnTable(t, sql, queryArgs, [][3]string{
		{"abc12345-0000", "no", "no"},        // ID が語で始まる → 当たる
		{"12345abc-0000", "no", "no"},        // ID の途中に語 → 当たらない
		{"ffff-0000", "title has ABC", "no"}, // 列に含む（大小無視）→ 当たる
		{"eeee-0000", "no", "no"},            // どこにも無い → 当たらない
	})

	want := map[string]bool{"abc12345-0000": true, "ffff-0000": true}
	if len(matchedIDs) != len(want) {
		t.Fatalf("ID は前方一致だけのはず: got %v, want %v", matchedIDs, want)
	}
	for _, id := range matchedIDs {
		if !want[id] {
			t.Errorf("一致してはいけない行が一致した: %q (matched=%v)", id, matchedIDs)
		}
	}
}

// 除外語は ID を見ない。`-1` で UUID に 1 を含む記録が消えていた事故の再発防止。
func TestGenerateFindSQLCommon_NotWordsDoNotLookAtID(t *testing.T) {
	query := &find.FindQuery{
		Words:    []string{},
		NotWords: []string{"abc"},
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE", "SHOP"}, true, false,
		false, true, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(sql, "(ID) NOT LIKE") {
		t.Errorf("除外語は ID 列を見てはいけない, got %q", sql)
	}

	matchedIDs := matchedIDsOfTwoColumnTable(t, sql, queryArgs, [][3]string{
		{"abc12345-0000", "keep", "keep"},   // ID が除外語で始まっても残る
		{"1111-0000", "has abc here", "no"}, // TITLE に除外語 → 消える
		{"2222-0000", "no", "shop ABC"},     // SHOP に除外語（大小無視）→ 消える
		{"3333-0000", "keep", "keep"},       // 残る
	})

	want := map[string]bool{"abc12345-0000": true, "3333-0000": true}
	if len(matchedIDs) != len(want) {
		t.Fatalf("除外語は対象列だけで判定するはず: got %v, want %v", matchedIDs, want)
	}
	for _, id := range matchedIDs {
		if !want[id] {
			t.Errorf("除外されるべき行が残った: %q (matched=%v)", id, matchedIDs)
		}
	}
}

// WordsSkipIDMatch が真なら肯定語でも ID を見ない（除外語を肯定語として再検索する内部クエリ用）。
func TestGenerateFindSQLCommon_WordsSkipIDMatch(t *testing.T) {
	query := &find.FindQuery{
		Words:            []string{"abc"},
		WordsAnd:         false,
		WordsSkipIDMatch: true,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE", "SHOP"}, true, false,
		false, true, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(sql, "(ID) LIKE") {
		t.Errorf("WordsSkipIDMatch のときは ID 列を見てはいけない, got %q", sql)
	}
	if len(queryArgs) != 2 {
		t.Errorf("バインド値は対象列2つぶんのはず, got %v", queryArgs)
	}

	matchedIDs := matchedIDsOfTwoColumnTable(t, sql, queryArgs, [][3]string{
		{"abc12345-0000", "no", "no"},  // ID が語で始まっても当たらない
		{"1111-0000", "has abc", "no"}, // 列に含む → 当たる
	})
	if len(matchedIDs) != 1 || matchedIDs[0] != "1111-0000" {
		t.Errorf("列だけで判定するはず: got %v", matchedIDs)
	}

	// 見る列が無く ID も見ないなら、何にも一致しない（素通しにしてはいけない）
	whereCounter = 0
	queryArgs = []any{}
	sql, err = GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{}, true, false,
		false, true, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "0 = 1") {
		t.Errorf("列も ID も見ないなら一致しえないはず, got %q", sql)
	}
	if err := assertValidWhereClause(t, sql, queryArgs); err != nil {
		t.Errorf("生成されたWHERE句がSQLiteで実行できない: %v (sql=%q)", err, sql)
	}
}

// 数値列（Nlog の AMOUNT / KC の NUM_VALUE / Lantana の MOOD）は列名を渡すだけで、
// SQLite の暗黙変換により文字列として部分一致する。TEXT 保存の値も INTEGER 保存の値も同じ。
func TestGenerateFindSQLCommon_NumericColumnMatchesAsText(t *testing.T) {
	query := &find.FindQuery{
		Words:    []string{"1500"},
		WordsAnd: false,
	}
	whereCounter := 0
	queryArgs := []any{}

	whereSQL, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE", "SHOP"}, true, false,
		false, true, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SHOP 列を数値列に見立てる。TEXT の "1500" と INTEGER の 1500 の両方を入れる
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("error at open memory db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE MY_TABLE (ID, TITLE, SHOP, RELATED_TIME, UPDATE_TIME)`); err != nil {
		t.Fatalf("error at create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO MY_TABLE (ID, TITLE, SHOP) VALUES ('text-1500', 'x', '1500'), ('int-1500', 'x', 1500), ('int-315', 'x', 315), ('real-1500', 'x', 1500.0)`); err != nil {
		t.Fatalf("error at insert: %v", err)
	}
	rows, err := db.Query(`SELECT ID FROM MY_TABLE AS T WHERE `+whereSQL, queryArgs...)
	if err != nil {
		t.Fatalf("error at select: %v", err)
	}
	defer func() { _ = rows.Close() }()
	matched := map[string]bool{}
	for rows.Next() {
		id := ""
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("error at scan: %v", err)
		}
		matched[id] = true
	}
	if !matched["text-1500"] || !matched["int-1500"] || !matched["real-1500"] {
		t.Errorf("数値列は保存型によらず文字列として当たるはず: got %v", matched)
	}
	if matched["int-315"] {
		t.Errorf("語を含まない数値は当たらないはず: got %v", matched)
	}
}
