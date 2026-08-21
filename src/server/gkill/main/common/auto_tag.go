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

// autoTagAppName はタグのCREATE_APP/UPDATE_APPへ刻む名前。
// 独立バイナリだった頃から変えていない。付与元をあとから見分けるための値なので、
// 変えると過去に付けたぶんと出所が食い違う。
const autoTagAppName = "gkill_auto_tag"

// autoTagLocaleName はAPIへ渡すロケール。エラーメッセージの取得にしか使わない。
const autoTagLocaleName = "ja"

// autoTagIDNamespace は自動付与したタグのIDを決めるための名前空間。
//
// 同じ(対象ID, タグ名)には常に同じIDを振る。これが冪等性の要で、
// 「付いているか」の判定を取りこぼしても、サーバ側が同じIDのタグを
// AlreadyExistTagErrorで弾くので二重登録にならない。
// 論理削除されたタグも同じIDで弾かれるため、消したタグが復活することもない。
//
// 文字列を変えると過去に付与したぶんとIDが食い違い、全件が付け直しになる。
var autoTagIDNamespace = uuid.NewSHA1(uuid.NameSpaceOID, []byte("github.com/mt3hr/gkill/"+autoTagAppName))

var (
	autoTagByRepPrefixArgs []string
	autoTagByRepNameArgs   []string
	autoTagDryRun          bool
)

// autoTagPrefixRule は「repの名前がPrefixで始まるならTagを付ける」ルール。
type autoTagPrefixRule struct {
	Prefix string
	Tag    string
}

// autoTagTarget はタグを付ける対象のKyouと、付けるタグ名。
type autoTagTarget struct {
	Kyou reps.Kyou
	Tags []string
}

// AutoTagCmd はリポジトリ単位のルールでKyouへタグを自動付与する。
//
// 判定も付与も稼働中のgkill_serverのAPI越しに行う。
// 認証はissueLocalSessionで発行する対象ユーザの短命セッション
// (APIはセッションのユーザとして動くので、管理者セッションでは対象ユーザのrepを見られない)。
//
// すでに同じタグが付いているKyouには何もしないので、何度実行してもよい。
var AutoTagCmd = &cobra.Command{
	Use:           "auto_tag",
	Short:         `auto_tag 'user_id'`,
	Args:          cobra.ArbitraryArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Usage()
		}
		ctx := cmd.Context()

		prefixRules, repTypeRules, err := parseAutoTagRules(autoTagByRepPrefixArgs, autoTagByRepNameArgs)
		if err != nil {
			// 指定の書式が誤り。usageを見せてからエラーで返す(exit 1)。
			cmd.Usage()
			return err
		}
		if len(prefixRules) == 0 && len(repTypeRules) == 0 {
			cmd.Usage()
			return errors.New("--tag_by_rep_prefix か --tag_by_rep_name を1つ以上指定してください")
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
			if err := autoTagForUser(ctx, endpoint, configDBRootDir, userID, prefixRules, repTypeRules); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	},
}

func init() {
	AutoTagCmd.Flags().StringArrayVar(&autoTagByRepPrefixArgs, "tag_by_rep_prefix", nil, "'<rep name prefix>=<tag>'")
	AutoTagCmd.Flags().StringArrayVar(&autoTagByRepNameArgs, "tag_by_rep_name", nil, "'<rep type>' (tag = rep name)")
	AutoTagCmd.Flags().BoolVar(&autoTagDryRun, "dry_run", false, "")
}

// parseAutoTagRules はコマンドライン引数をルールへ変換する。
func parseAutoTagRules(prefixArgs []string, repTypeArgs []string) ([]autoTagPrefixRule, []string, error) {
	prefixRules := []autoTagPrefixRule{}
	for _, arg := range prefixArgs {
		prefix, tag, found := strings.Cut(arg, "=")
		if !found || prefix == "" || tag == "" {
			return nil, nil, fmt.Errorf("--tag_by_rep_prefix は '<rep name prefix>=<tag>' の形で指定してください: %s", arg)
		}
		prefixRules = append(prefixRules, autoTagPrefixRule{Prefix: prefix, Tag: tag})
	}

	repTypeRules := []string{}
	for _, repType := range repTypeArgs {
		if repType == "" {
			return nil, nil, fmt.Errorf("--tag_by_rep_name にrepの種別を指定してください")
		}
		repTypeRules = append(repTypeRules, repType)
	}
	return prefixRules, repTypeRules, nil
}

// autoTagForUser は1ユーザぶんのタグ付与を行う。
func autoTagForUser(ctx context.Context, endpoint *LocalServerEndpoint, configDBRootDir string, userID string, prefixRules []autoTagPrefixRule, repTypeRules []string) error {
	sessionID, refreshSession, cleanupSession, err := issueLocalSession(ctx, configDBRootDir, endpoint.Device, userID)
	if err != nil {
		return fmt.Errorf("error at issue local session user id = %s: %w", userID, err)
	}
	defer cleanupSession()

	client := &autoTagAPIClient{Endpoint: endpoint, SessionID: sessionID}
	targets := map[string]*autoTagTarget{}

	for _, rule := range prefixRules {
		if err := collectByRepPrefix(ctx, client, userID, rule, targets); err != nil {
			return err
		}
	}
	for _, repType := range repTypeRules {
		if err := collectByRepType(ctx, client, userID, repType, targets); err != nil {
			return err
		}
	}

	if len(targets) == 0 {
		fmt.Printf("%s: no target\n", userID)
		return nil
	}
	return addAutoTags(ctx, client, userID, targets, refreshSession)
}

// autoTagRefreshInterval は、この件数タグを付けるごとにセッションのTTLを延長する間隔。
// 進捗印字と同じ区切り。大量付与でセッションが期限切れになるのを防ぐ。
const autoTagRefreshInterval = 500

// shouldRefreshAutoTagSession は、これまでに付けた件数addedがrefresh間隔の区切りかを返す。
// added==0(まだ1件も付けていない)では延長しない。
func shouldRefreshAutoTagSession(added int) bool {
	return added > 0 && added%autoTagRefreshInterval == 0
}

// collectByRepPrefix は接頭辞ルールの対象を集める。
//
// rep名の一覧をサーバから取り、接頭辞に一致したrepを対象にする。
// 「全件」と「そのタグが付いている件」の2回の検索の差分が、まだ付いていないKyou。
func collectByRepPrefix(ctx context.Context, client *autoTagAPIClient, userID string, rule autoTagPrefixRule, targets map[string]*autoTagTarget) error {
	repNames, err := client.GetAllRepNames(ctx)
	if err != nil {
		return fmt.Errorf("error at get all rep names user id = %s: %w", userID, err)
	}

	matchRepNames := []string{}
	for _, repName := range repNames {
		if strings.HasPrefix(repName, rule.Prefix) {
			matchRepNames = append(matchRepNames, repName)
		}
	}
	if len(matchRepNames) == 0 {
		fmt.Printf("%s: no rep matched: prefix = %s\n", userID, rule.Prefix)
		return nil
	}
	slices.Sort(matchRepNames)

	allKyous, err := client.GetKyous(ctx, &find.FindQuery{Reps: matchRepNames})
	if err != nil {
		return fmt.Errorf("error at find kyous prefix = %s user id = %s: %w", rule.Prefix, userID, err)
	}
	taggedIDs, err := client.FindTaggedKyouIDs(ctx, &find.FindQuery{Reps: matchRepNames}, rule.Tag)
	if err != nil {
		return fmt.Errorf("error at find tagged kyous prefix = %s user id = %s: %w", rule.Prefix, userID, err)
	}

	added := 0
	for _, kyou := range allKyous {
		if _, exist := taggedIDs[kyou.ID]; exist {
			continue
		}
		addAutoTagTarget(targets, kyou, rule.Tag)
		added++
	}
	fmt.Printf("%s: prefix = %s tag = %s reps = %d (%s) kyous = %d tagged = %d target = %d\n",
		userID, rule.Prefix, rule.Tag, len(matchRepNames), strings.Join(matchRepNames, ", "), len(allKyous), len(taggedIDs), added)
	return nil
}

// collectByRepType はrep種別ルールの対象を集める。タグ名はKyouが入っているrepの名前そのもの。
//
// タグ名がrepごとに違うので、付与済みの照会もrep名ごとに分けて投げる。
func collectByRepType(ctx context.Context, client *autoTagAPIClient, userID string, repType string, targets map[string]*autoTagTarget) error {
	allKyous, err := client.GetKyous(ctx, &find.FindQuery{RepTypes: []string{repType}})
	if err != nil {
		return fmt.Errorf("error at find kyous rep type = %s user id = %s: %w", repType, userID, err)
	}

	kyousByRepName := map[string][]reps.Kyou{}
	for _, kyou := range allKyous {
		if kyou.RepName == "" {
			continue
		}
		kyousByRepName[kyou.RepName] = append(kyousByRepName[kyou.RepName], kyou)
	}
	if len(kyousByRepName) == 0 {
		fmt.Printf("%s: no rep matched: rep type = %s\n", userID, repType)
		return nil
	}

	repNames := make([]string, 0, len(kyousByRepName))
	for repName := range kyousByRepName {
		repNames = append(repNames, repName)
	}
	slices.Sort(repNames)

	added, tagged := 0, 0
	for _, repName := range repNames {
		query := &find.FindQuery{
			RepTypes: []string{repType},
			Reps:     []string{repName},
		}
		taggedIDs, err := client.FindTaggedKyouIDs(ctx, query, repName)
		if err != nil {
			return fmt.Errorf("error at find tagged kyous rep = %s user id = %s: %w", repName, userID, err)
		}
		tagged += len(taggedIDs)
		for _, kyou := range kyousByRepName[repName] {
			if _, exist := taggedIDs[kyou.ID]; exist {
				continue
			}
			addAutoTagTarget(targets, kyou, repName)
			added++
		}
	}
	fmt.Printf("%s: rep type = %s reps = %d kyous = %d tagged = %d target = %d\n",
		userID, repType, len(repNames), len(allKyous), tagged, added)
	return nil
}

// addAutoTagTarget は付与予定へ1件積む。同じタグを二度積まない。
func addAutoTagTarget(targets map[string]*autoTagTarget, kyou reps.Kyou, tagName string) {
	target, exist := targets[kyou.ID]
	if !exist {
		target = &autoTagTarget{Kyou: kyou}
		targets[kyou.ID] = target
	}
	if slices.Contains(target.Tags, tagName) {
		return
	}
	target.Tags = append(target.Tags, tagName)
}

// autoTagID は(対象ID, タグ名)から決まるタグのIDを返す。
func autoTagID(targetID string, tagName string) string {
	return uuid.NewSHA1(autoTagIDNamespace, []byte(targetID+"\x00"+tagName)).String()
}

// addAutoTags は集めた対象へタグを付ける。
// refresh は500件ごとの進捗印字に相乗りしてセッションのTTLを延ばす(長時間実行での期限切れ防止)。
func addAutoTags(ctx context.Context, client *autoTagAPIClient, userID string, targets map[string]*autoTagTarget, refresh func() error) error {
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
				ID:        autoTagID(targetID, tagName),
				TargetID:  targetID,
				Tag:       tagName,
				// タグの関連時刻は対象のKyouの時刻に合わせる
				RelatedTime:  target.Kyou.RelatedTime,
				CreateTime:   runAt,
				CreateApp:    autoTagAppName,
				CreateDevice: client.Endpoint.Device,
				CreateUser:   userID,
				UpdateTime:   runAt,
				UpdateApp:    autoTagAppName,
				UpdateDevice: client.Endpoint.Device,
				UpdateUser:   userID,
			}

			if autoTagDryRun {
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
			if shouldRefreshAutoTagSession(added) {
				fmt.Printf("%s: added %d tags\n", userID, added)
				// セッションのTTLを延長する。失敗しても付与は続ける(次の区切りで挽回できる)。
				if refresh != nil {
					if err := refresh(); err != nil {
						slog.Log(ctx, gkill_log.Warn, "error at refresh auto_tag session ttl", "user_id", fmt.Sprintf("%q", userID), "error", fmt.Sprintf("%q", err))
					}
				}
			}
		}
	}

	fmt.Printf("%s: added = %d already_exist = %d elapsed = %s\n", userID, added, alreadyExist, time.Since(runAt).String())
	return nil
}

// autoTagAPIClient は稼働中のgkill_serverのAPIを叩く。
type autoTagAPIClient struct {
	Endpoint  *LocalServerEndpoint
	SessionID string
}

// post はJSONをPOSTして、レスポンスをresponseへ書き込む。
func (c *autoTagAPIClient) post(ctx context.Context, path string, requestBody any, response any) error {
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("error at post %s: status = %d body = %q", address, resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("error at decode response of %s: %w", address, err)
	}
	return nil
}

// GetAllRepNames はrep名の一覧を取る。
func (c *autoTagAPIClient) GetAllRepNames(ctx context.Context) ([]string, error) {
	response := &req_res.GetAllRepNamesResponse{}
	err := c.post(ctx, "/api/get_all_rep_names", &req_res.GetAllRepNamesRequest{
		SessionID:  c.SessionID,
		LocaleName: autoTagLocaleName,
	}, response)
	if err != nil {
		return nil, err
	}
	if err := autoTagResponseError("/api/get_all_rep_names", response.Errors); err != nil {
		return nil, err
	}
	return response.RepNames, nil
}

// GetKyous は検索条件に一致するKyouを取る。
func (c *autoTagAPIClient) GetKyous(ctx context.Context, query *find.FindQuery) ([]reps.Kyou, error) {
	response := &req_res.GetKyousResponse{}
	err := c.post(ctx, "/api/get_kyous", &req_res.GetKyousRequest{
		SessionID:  c.SessionID,
		Query:      query,
		LocaleName: autoTagLocaleName,
	}, response)
	if err != nil {
		return nil, err
	}
	if err := autoTagResponseError("/api/get_kyous", response.Errors); err != nil {
		return nil, err
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
// 渡されたqueryは書き換えない。rep名ごとに同じ条件を使い回すため。
func (c *autoTagAPIClient) buildTaggedQuery(query *find.FindQuery, tagName string) *find.FindQuery {
	taggedQuery := *query
	taggedQuery.Tags = []string{tagName}
	taggedQuery.TagsAnd = true
	return &taggedQuery
}

// FindTaggedKyouIDs は、渡した条件のうちtagNameが付いているKyouのIDを返す。
func (c *autoTagAPIClient) FindTaggedKyouIDs(ctx context.Context, query *find.FindQuery, tagName string) (map[string]struct{}, error) {
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
func (c *autoTagAPIClient) AddTag(ctx context.Context, tag reps.Tag) (alreadyExist bool, err error) {
	response := &req_res.AddTagResponse{}
	err = c.post(ctx, "/api/add_tag", &req_res.AddTagRequest{
		SessionID:  c.SessionID,
		Tag:        tag,
		LocaleName: autoTagLocaleName,
	}, response)
	if err != nil {
		return false, err
	}
	for _, gkillError := range response.Errors {
		if gkillError != nil && gkillError.ErrorCode == message.AlreadyExistTagError {
			return true, nil
		}
	}
	if err := autoTagResponseError("/api/add_tag", response.Errors); err != nil {
		return false, err
	}
	return false, nil
}

// autoTagResponseError は応答のerrorsを1つのerrorにまとめる。
// gkillはHTTP 200でもerrorsに中身を入れることがあるので、必ず見る。
func autoTagResponseError(path string, gkillErrors []*message.GkillError) error {
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
