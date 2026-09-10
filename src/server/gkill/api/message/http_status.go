package message

import "net/http"

// HTTPStatusOf は GkillError のエラーコードに対応する HTTP ステータスを返します。
//
// 表に無いコードは 0 を返します。0 は「まだ分類していない」の意味で、
// http_status_test.go が error_codes.go をソース走査して 0 のコードを落とすので、
// エラーコードを足した人は必ずここへも1行足すことになります。
//
// 分類は名前の規則から機械的には導けません。error_codes.go の語彙には
// Forbidden / Unauthorized / Denied / Permission が1件も無く、Invalid* が 400 と 500 に
// 跨り、NotFound* が 401 と 404 に跨るためです。だから推論ではなく表にしてあります。
//
// **Invalid*ResponseDataError（応答のエンコード失敗）だけは、割り当てた 500 が実際には効きません。**
// あのコードが積まれるのは json.NewEncoder(w).Encode(response) が失敗したときで、
// その時点ではステータス行がもう送出済みだからです。表に入れてあるのは網羅性のためです。
func HTTPStatusOf(errorCode string) int {
	return errorCodeHTTPStatus[errorCode]
}

// HTTPStatusForErrors は errors 配列全体から応答の HTTP ステータスを決めます。
//
// 空(または nil)なら 200。複数あるときは statusPriority の順で最も重いものを選びます。
// 表に無いコードは 500 として扱います(分類漏れを 200 で握り潰さないため)。
func HTTPStatusForErrors(errs []*GkillError) int {
	best := http.StatusOK
	bestRank := -1
	for _, err := range errs {
		if err == nil {
			continue
		}
		status := errorCodeHTTPStatus[err.ErrorCode]
		if status == 0 {
			status = http.StatusInternalServerError
		}
		rank := statusRank(status)
		if rank > bestRank {
			best, bestRank = status, rank
		}
	}
	return best
}

// statusPriority は複数のエラーが載ったときにどれを応答のステータスにするかの優先順です。
//
// サーバの障害(500)を利用者側の誤り(4xx)で覆い隠さないことを最優先にし、
// 次に「まず認証を直せ」(401) → 「認証は通っているが許可されない」(403) の順にしています。
// 残りは、利用者が次に取る行動が明確なものほど後ろです。
var statusPriority = []int{
	http.StatusInternalServerError,
	http.StatusUnauthorized,
	http.StatusForbidden,
	http.StatusTooManyRequests,
	http.StatusConflict,
	http.StatusNotFound,
	http.StatusBadRequest,
	// 413 は 400 の特化形で、「本文を小さくする」という次の行動が最も明確なので最後尾。
	// (実際には認証系ミドルウェアがハンドラより手前で単独エラーとして返すので、
	// 他のエラーと同居することは今のところ無い)
	http.StatusRequestEntityTooLarge,
}

// statusRank は statusPriority の中での重さを返します。大きいほど重い。
func statusRank(status int) int {
	for i, s := range statusPriority {
		if s == status {
			return len(statusPriority) - i
		}
	}
	return 0
}

// errorCodeHTTPStatus はエラーコードと HTTP ステータスの対応表(正本)です。
//
// キーは error_codes.go の定数そのものです(定数の値が ERR 番号なので、
// 定数名で書いておけばリネームに追随します)。行末のコメントは grep 用の番号です。
var errorCodeHTTPStatus = map[string]int{
	// ---- 400 Bad Request — リクエストの作りが悪い（利用者・呼び出し側で直せる） ----
	// ほとんどが「リクエストJSONのパースに失敗した」。トークン不正・期限切れもここ。
	AccountInvalidLoginRequestDataError:                             http.StatusBadRequest, // ERR000001
	AccountInvalidLogoutRequestDataError:                            http.StatusBadRequest, // ERR000008
	AccountInvalidResetPasswordRequestDataError:                     http.StatusBadRequest, // ERR000012
	AccountInvalidSetNewPasswordRequestDataError:                    http.StatusBadRequest, // ERR000016
	InvalidAddTagRequestDataError:                                   http.StatusBadRequest, // ERR000025
	InvalidAddTextRequestDataError:                                  http.StatusBadRequest, // ERR000027
	InvalidAddURLogRequestDataError:                                 http.StatusBadRequest, // ERR000031
	InvalidAddNlogRequestDataError:                                  http.StatusBadRequest, // ERR000035
	InvalidAddTimeIsRequestDataError:                                http.StatusBadRequest, // ERR000039
	InvalidAddLantanaRequestDataError:                               http.StatusBadRequest, // ERR000043
	InvalidAddKyouInfoRequestDataError:                              http.StatusBadRequest, // ERR000047
	InvalidAddReKyouRequestDataError:                                http.StatusBadRequest, // ERR000051
	InvalidUpdateTagRequestDataError:                                http.StatusBadRequest, // ERR000053
	InvalidUpdateTextRequestDataError:                               http.StatusBadRequest, // ERR000065
	InvalidUpdateKmemoRequestDataError:                              http.StatusBadRequest, // ERR000068
	InvalidUpdateURLogRequestDataError:                              http.StatusBadRequest, // ERR000071
	InvalidUpdateNlogRequestDataError:                               http.StatusBadRequest, // ERR000074
	InvalidUpdateTimeIsRequestDataError:                             http.StatusBadRequest, // ERR000077
	InvalidUpdateLantanaRequestDataError:                            http.StatusBadRequest, // ERR000080
	InvalidAddMiRequestDataError:                                    http.StatusBadRequest, // ERR000085
	InvalidUpdateMiRequestDataError:                                 http.StatusBadRequest, // ERR000088
	InvalidUpdateKyouInfoRequestDataError:                           http.StatusBadRequest, // ERR000090
	InvalidUpdateReKyouRequestDataError:                             http.StatusBadRequest, // ERR000093
	AccountInvalidAddKmemoRequestDataError:                          http.StatusBadRequest, // ERR000097
	InvalidGetKyousRequestDataError:                                 http.StatusBadRequest, // ERR000099
	InvalidGetKyouRequestDataError:                                  http.StatusBadRequest, // ERR000100
	InvalidGetKmemoRequestDataError:                                 http.StatusBadRequest, // ERR000103
	InvalidGetURLogRequestDataError:                                 http.StatusBadRequest, // ERR000105
	InvalidGetNlogRequestDataError:                                  http.StatusBadRequest, // ERR000107
	InvalidGetTimeIsRequestDataError:                                http.StatusBadRequest, // ERR000109
	InvalidGetMiRequestDataError:                                    http.StatusBadRequest, // ERR000111
	InvalidGetLantanaRequestDataError:                               http.StatusBadRequest, // ERR000113
	InvalidGetReKyouRequestDataError:                                http.StatusBadRequest, // ERR000115
	InvalidGetGitCommitLogRequestDataError:                          http.StatusBadRequest, // ERR000117
	InvalidGetGitCommitLogsRequestDataError:                         http.StatusBadRequest, // ERR000120
	InvalidGetMiBoardNamesRequestDataError:                          http.StatusBadRequest, // ERR000123
	InvalidGetAllTagNamesRequestDataError:                           http.StatusBadRequest, // ERR000126
	InvalidGetTagsByTargetIDRequestDataError:                        http.StatusBadRequest, // ERR000129
	InvalidGetTagHistoriesByTagIDRequestDataError:                   http.StatusBadRequest, // ERR000132
	InvalidGetTextHistoriesByTextIDRequestDataError:                 http.StatusBadRequest, // ERR000135
	InvalidGetTextsByTargetIDRequestDataError:                       http.StatusBadRequest, // ERR000138
	InvalidGetApplicationConfigRequestDataError:                     http.StatusBadRequest, // ERR000141
	InvalidGetServerConfigRequestDataError:                          http.StatusBadRequest, // ERR000144
	InvalidUploadFilesRequestDataError:                              http.StatusBadRequest, // ERR000147
	InvalidUploadGPSLogFilesRequestDataError:                        http.StatusBadRequest, // ERR000149
	InvalidUpdateTagStructRequestDataError:                          http.StatusBadRequest, // ERR000162
	InvalidUpdateRepStructRequestDataError:                          http.StatusBadRequest, // ERR000167
	InvalidUpdateDeviceStructRequestDataError:                       http.StatusBadRequest, // ERR000172
	InvalidUpdateRepTypeStructRequestDataError:                      http.StatusBadRequest, // ERR000177
	InvalidUpdateAccountStatusRequestDataError:                      http.StatusBadRequest, // ERR000179
	InvalidUpdateUserRepsRequestDataError:                           http.StatusBadRequest, // ERR000182
	AccountInvalidAddAccountRequestDataError:                        http.StatusBadRequest, // ERR000185
	InvalidGetGPSLogRequestDataError:                                http.StatusBadRequest, // ERR000190
	InvalidGetGkillInfoRequestDataError:                             http.StatusBadRequest, // ERR000193
	InvalidGetShareKyouListInfosRequestDataError:                    http.StatusBadRequest, // ERR000195
	InvalidDeleteShareKyouListInfosRequestDataError:                 http.StatusBadRequest, // ERR000198
	InvalidGetMiSharedTasksRequestDataError:                         http.StatusBadRequest, // ERR000201
	InvalidGetKFTLTemplateRequestDataError:                          http.StatusBadRequest, // ERR000210
	InvalidAddShareKyouListInfoRequestDataError:                     http.StatusBadRequest, // ERR000212
	AccountInvalidGenerateTLSFileRequestDataError:                   http.StatusBadRequest, // ERR000217
	InvalidUpdateApplicationConfigRequestDataError:                  http.StatusBadRequest, // ERR000225
	InvalidUpdateKFTLTemplateRequestDataError:                       http.StatusBadRequest, // ERR000228
	InvalidUpdateServerConfigRequestDataError:                       http.StatusBadRequest, // ERR000233
	InvalidGetRepositoriesRequestDataError:                          http.StatusBadRequest, // ERR000241
	InvalidGetAllRepNamesRequestDataError:                           http.StatusBadRequest, // ERR000244
	InvalidPasswordResetTokenError:                                  http.StatusBadRequest, // ERR000247
	InvalidGetIDFKyouRequestDataError:                               http.StatusBadRequest, // ERR000249
	InvalidUpdateIDFKyouRequestDataError:                            http.StatusBadRequest, // ERR000252
	InvalidUpdateShareKyouListInfoRequestDataError:                  http.StatusBadRequest, // ERR000257
	InvalidGetPlayingTimeIsRequestDataError:                         http.StatusBadRequest, // ERR000265
	InvalidGetMiTaskNotificationPublicKeyRequestDataError:           http.StatusBadRequest, // ERR000268
	InvalidRegisterMiTaskNotificationRequest:                        http.StatusBadRequest, // ERR000270
	InvalidAddNotificationRequestDataError:                          http.StatusBadRequest, // ERR000274
	InvalidUpdateNotificationRequestDataError:                       http.StatusBadRequest, // ERR000279
	InvalidGetNotificationsByTargetIDRequestDataError:               http.StatusBadRequest, // ERR000283
	InvalidGetNotificationHistoriesByNotificationIDRequestDataError: http.StatusBadRequest, // ERR000287
	InvalidRegisterOpenDirectoryRequest:                             http.StatusBadRequest, // ERR000292
	InvalidRegisterOpenFileRequest:                                  http.StatusBadRequest, // ERR000294
	InvalidUpdateDnoteJSONDataRequestDataError:                      http.StatusBadRequest, // ERR000302
	InvalidUpdateKCRequestDataError:                                 http.StatusBadRequest, // ERR000308
	AccountInvalidAddKCRequestDataError:                             http.StatusBadRequest, // ERR000312
	InvalidGetKCRequestDataError:                                    http.StatusBadRequest, // ERR000314
	InvalidGetUpdatedDatasByTimeRequest:                             http.StatusBadRequest, // ERR000316
	AccountInvalidCommitTxRequestDataError:                          http.StatusBadRequest, // ERR000319
	AccountInvalidDiscardTxRequestDataError:                         http.StatusBadRequest, // ERR000333
	InvalidSubmitKFTLTextRequestDataError:                           http.StatusBadRequest, // ERR000350
	SubmitKFTLTextInvalidInputError:                                 http.StatusBadRequest, // ERR000416
	InvalidGetKyousMCPRequestDataError:                              http.StatusBadRequest, // ERR000352
	InvalidUpdateCacheRequestDataError:                              http.StatusBadRequest, // ERR000355
	InvalidURLogBookmarkletRequestDataError:                         http.StatusBadRequest, // ERR000372
	InvalidBrowseZipContentsRequestDataError:                        http.StatusBadRequest, // ERR000375
	InvalidGetPluginListRequestDataError:                            http.StatusBadRequest, // ERR000377
	InvalidGetPluginContentHTMLRequestDataError:                     http.StatusBadRequest, // ERR000379
	InvalidGetPluginConfigHTMLRequestDataError:                      http.StatusBadRequest, // ERR000381
	InvalidPostPluginConfigRequestDataError:                         http.StatusBadRequest, // ERR000383
	InvalidGetIDFKyouByRelativePathRequestDataError:                 http.StatusBadRequest, // ERR000385
	InvalidAddMiReKyouRequestDataError:                              http.StatusBadRequest, // ERR000390
	InvalidUpdateMiReKyouRequestDataError:                           http.StatusBadRequest, // ERR000394
	InvalidGetMiReKyouRequestDataError:                              http.StatusBadRequest, // ERR000398
	InvalidGetReKyousByTargetIDRequestDataError:                     http.StatusBadRequest, // ERR000402
	InvalidGetMiReKyousByTargetIDRequestDataError:                   http.StatusBadRequest, // ERR000405
	ExpiredPasswordResetTokenError:                                  http.StatusBadRequest, // ERR000408
	InvalidGetRepInfosMCPRequestDataError:                           http.StatusBadRequest, // ERR000411

	// ---- 401 Unauthorized — 誰なのか確認できない ----
	// クライアントの check_auth と MCP の再ログインがこの4つを契機にしている。
	AccountNotFoundError:        http.StatusUnauthorized, // ERR000002
	AccountInvalidPasswordError: http.StatusUnauthorized, // ERR000005
	AccountSessionNotFoundError: http.StatusUnauthorized, // ERR000013
	AccountSessionExpiredError:  http.StatusUnauthorized, // ERR000373

	// ---- 403 Forbidden — 誰かは分かるが、やらせない ----
	// 権限不足・無効化済みアカウント・ローカル限定アクセス違反。
	AccountIsNotEnableError:         http.StatusForbidden, // ERR000003
	AccountNotHasAdminError:         http.StatusForbidden, // ERR000014
	TagStructInvalidUserID:          http.StatusForbidden, // ERR000159
	RepStructInvalidUserID:          http.StatusForbidden, // ERR000164
	DeviceStructInvalidUserID:       http.StatusForbidden, // ERR000169
	RepTypeStructInvalidUserID:      http.StatusForbidden, // ERR000174
	KFTLTemplateStructInvalidUserID: http.StatusForbidden, // ERR000229
	AccountDisabledError:            http.StatusForbidden, // ERR000238
	OpenFolderNotLocalAccountError:  http.StatusForbidden, // ERR000296
	CannotDisableOwnAccountError:    http.StatusForbidden, // ERR000409
	LocalOnlyAccessDeniedError:      http.StatusForbidden, // ERR000414

	// ---- 404 Not Found — 指定されたものが無い ----
	// 「サーバの設定ファイルが無い」は利用者の指定ミスではないので 500 に置いてある。
	NotFoundTagError:               http.StatusNotFound, // ERR000055
	NotFoundTextError:              http.StatusNotFound, // ERR000067
	NotFoundKmemoError:             http.StatusNotFound, // ERR000070
	NotFoundURLogError:             http.StatusNotFound, // ERR000073
	NotFoundNlogError:              http.StatusNotFound, // ERR000076
	NotFoundTimeIsError:            http.StatusNotFound, // ERR000079
	NotFoundLantanaError:           http.StatusNotFound, // ERR000082
	NotFoundMiError:                http.StatusNotFound, // ERR000089
	NotFoundKyouInfoError:          http.StatusNotFound, // ERR000092
	NotFoundReKyouError:            http.StatusNotFound, // ERR000095
	NotFoundTargetIDFRepError:      http.StatusNotFound, // ERR000151
	NotFoundTargetGPSLogRepError:   http.StatusNotFound, // ERR000153
	NotFoundIDFKyouError:           http.StatusNotFound, // ERR000254
	NotExistShareKyouListInfoError: http.StatusNotFound, // ERR000259
	NotFoundNotificationError:      http.StatusNotFound, // ERR000280
	NotFoundKCError:                http.StatusNotFound, // ERR000310
	NotFoundMiReKyouError:          http.StatusNotFound, // ERR000397
	TargetAccountNotFoundError:     http.StatusNotFound, // ERR000413

	// ---- 409 Conflict — 今の状態と衝突する ----
	// 同じIDが既にある、リセット中のアカウントにログインしようとした、など。
	AccountPasswordResetTokenIsNotNilError: http.StatusConflict, // ERR000004
	AlreadyExistTagError:                   http.StatusConflict, // ERR000056
	AlreadyExistTextError:                  http.StatusConflict, // ERR000057
	AlreadyExistKmemoError:                 http.StatusConflict, // ERR000058
	AlreadyExistURLogError:                 http.StatusConflict, // ERR000059
	AlreadyExistNlogError:                  http.StatusConflict, // ERR000060
	AlreadyExistTimeIsError:                http.StatusConflict, // ERR000061
	AlreadyExistLantanaError:               http.StatusConflict, // ERR000062
	AlreadyExistKyouInfoError:              http.StatusConflict, // ERR000063
	AlreadyExistReKyouError:                http.StatusConflict, // ERR000064
	AlreadyExistMiError:                    http.StatusConflict, // ERR000087
	AlreadyExistAccountError:               http.StatusConflict, // ERR000187
	AlreadyExistShareKyouListInfoError:     http.StatusConflict, // ERR000214
	AlreadyExistNotificationError:          http.StatusConflict, // ERR000276
	AlreadyExistKCError:                    http.StatusConflict, // ERR000307
	AlreadyExistMiReKyouError:              http.StatusConflict, // ERR000393

	// ---- 413 Request Entity Too Large — リクエスト本文が大きすぎる ----
	// 認証系ミドルウェアの先読み上限（maxAuthBodyBytes）超過。
	RequestBodyTooLargeError: http.StatusRequestEntityTooLarge, // ERR000417

	// ---- 429 Too Many Requests ----
	// ログインのレート制限（IP毎15分10回）。
	LoginRateLimitError: http.StatusTooManyRequests, // ERR000374

	// ---- 500 Internal Server Error — サーバ側の失敗 ----
	// 取得・追加・更新・削除の処理失敗と、応答エンコード失敗。分類の既定値でもある。
	AccountLoginInternalServerError:                                  http.StatusInternalServerError, // ERR000006
	AccountInvalidLoginResponseDataError:                             http.StatusInternalServerError, // ERR000007
	AccountInvalidLogoutResponseDataError:                            http.StatusInternalServerError, // ERR000009
	AccountLogoutInternalServerError:                                 http.StatusInternalServerError, // ERR000010
	AccountInvalidResetPasswordResponseDataError:                     http.StatusInternalServerError, // ERR000011
	AccountInfoUpdateError:                                           http.StatusInternalServerError, // ERR000015
	AccountInvalidSetNewPasswordResponseDataError:                    http.StatusInternalServerError, // ERR000017
	RepositoriesGetError:                                             http.StatusInternalServerError, // ERR000018
	AddTagError:                                                      http.StatusInternalServerError, // ERR000019
	GetTagError:                                                      http.StatusInternalServerError, // ERR000020
	AddTextError:                                                     http.StatusInternalServerError, // ERR000021
	GetTextError:                                                     http.StatusInternalServerError, // ERR000022
	AddKmemoError:                                                    http.StatusInternalServerError, // ERR000023
	GetKmemoError:                                                    http.StatusInternalServerError, // ERR000024
	InvalidAddTagResponseDataError:                                   http.StatusInternalServerError, // ERR000026
	InvalidAddTextResponseDataError:                                  http.StatusInternalServerError, // ERR000028
	AddURLogError:                                                    http.StatusInternalServerError, // ERR000029
	GetURLogError:                                                    http.StatusInternalServerError, // ERR000030
	InvalidAddURLogResponseDataError:                                 http.StatusInternalServerError, // ERR000032
	AddNlogError:                                                     http.StatusInternalServerError, // ERR000033
	GetNlogError:                                                     http.StatusInternalServerError, // ERR000034
	InvalidAddNlogResponseDataError:                                  http.StatusInternalServerError, // ERR000036
	AddTimeIsError:                                                   http.StatusInternalServerError, // ERR000037
	GetTimeIsError:                                                   http.StatusInternalServerError, // ERR000038
	InvalidAddTimeIsResponseDataError:                                http.StatusInternalServerError, // ERR000040
	AddLantanaError:                                                  http.StatusInternalServerError, // ERR000041
	GetLantanaError:                                                  http.StatusInternalServerError, // ERR000042
	InvalidAddLantanaResponseDataError:                               http.StatusInternalServerError, // ERR000044
	AddKyouInfoError:                                                 http.StatusInternalServerError, // ERR000045
	GetKyouInfoError:                                                 http.StatusInternalServerError, // ERR000046
	InvalidAddKyouInfoResponseDataError:                              http.StatusInternalServerError, // ERR000048
	AddReKyouError:                                                   http.StatusInternalServerError, // ERR000049
	GetReKyouError:                                                   http.StatusInternalServerError, // ERR000050
	InvalidAddReKyouResponseDataError:                                http.StatusInternalServerError, // ERR000052
	InvalidUpdateTagResponseDataError:                                http.StatusInternalServerError, // ERR000054
	InvalidUpdateTextResponseDataError:                               http.StatusInternalServerError, // ERR000066
	InvalidUpdateKmemoResponseDataError:                              http.StatusInternalServerError, // ERR000069
	InvalidUpdateURLogResponseDataError:                              http.StatusInternalServerError, // ERR000072
	InvalidUpdateNlogResponseDataError:                               http.StatusInternalServerError, // ERR000075
	InvalidUpdateTimeIsResponseDataError:                             http.StatusInternalServerError, // ERR000078
	InvalidUpdateLantanaResponseDataError:                            http.StatusInternalServerError, // ERR000081
	AddMiError:                                                       http.StatusInternalServerError, // ERR000083
	GetMiError:                                                       http.StatusInternalServerError, // ERR000084
	InvalidAddMiResponseDataError:                                    http.StatusInternalServerError, // ERR000086
	InvalidUpdateKyouInfoResponseDataError:                           http.StatusInternalServerError, // ERR000091
	InvalidUpdateReKyouResponseDataError:                             http.StatusInternalServerError, // ERR000094
	AccountInvalidAddKmemoResponseDataError:                          http.StatusInternalServerError, // ERR000096
	InvalidGetKyousResponseDataError:                                 http.StatusInternalServerError, // ERR000098
	GetKyouError:                                                     http.StatusInternalServerError, // ERR000101
	InvalidGetKmemoResponseDataError:                                 http.StatusInternalServerError, // ERR000102
	InvalidGetURLogResponseDataError:                                 http.StatusInternalServerError, // ERR000104
	InvalidGetNlogResponseDataError:                                  http.StatusInternalServerError, // ERR000106
	InvalidGetTimeIsResponseDataError:                                http.StatusInternalServerError, // ERR000108
	InvalidGetMiResponseDataError:                                    http.StatusInternalServerError, // ERR000110
	InvalidGetLantanaResponseDataError:                               http.StatusInternalServerError, // ERR000112
	InvalidGetReKyouResponseDataError:                                http.StatusInternalServerError, // ERR000114
	InvalidGetGitCommitLogResponseDataError:                          http.StatusInternalServerError, // ERR000116
	GetGitCommitLogError:                                             http.StatusInternalServerError, // ERR000118
	InvalidGetGitCommitLogsResponseDataError:                         http.StatusInternalServerError, // ERR000119
	GetGitCommitLogsError:                                            http.StatusInternalServerError, // ERR000121
	InvalidGetMiBoardNamesResponseDataError:                          http.StatusInternalServerError, // ERR000122
	GetMiBoardNamesError:                                             http.StatusInternalServerError, // ERR000124
	InvalidGetAllTagNamesResponseDataError:                           http.StatusInternalServerError, // ERR000125
	GetAllTagNamesError:                                              http.StatusInternalServerError, // ERR000127
	InvalidGetTagsByTargetIDResponseDataError:                        http.StatusInternalServerError, // ERR000128
	GetTagsByTargetIDError:                                           http.StatusInternalServerError, // ERR000130
	InvalidGetTagHistoriesByTagIDResponseDataError:                   http.StatusInternalServerError, // ERR000131
	GetTagHistoriesByTagIDError:                                      http.StatusInternalServerError, // ERR000133
	InvalidGetTextHistoriesByTextIDResponseDataError:                 http.StatusInternalServerError, // ERR000134
	GetTextHistoriesByTextIDError:                                    http.StatusInternalServerError, // ERR000136
	InvalidGetTextsByTargetIDResponseDataError:                       http.StatusInternalServerError, // ERR000137
	GetTextsByTargetIDError:                                          http.StatusInternalServerError, // ERR000139
	InvalidGetApplicationConfigResponseDataError:                     http.StatusInternalServerError, // ERR000140
	GetApplicationConfigError:                                        http.StatusInternalServerError, // ERR000142
	InvalidGetServerConfigResponseDataError:                          http.StatusInternalServerError, // ERR000143
	GetServerConfigError:                                             http.StatusInternalServerError, // ERR000145
	InvalidUploadFilesResponseDataError:                              http.StatusInternalServerError, // ERR000146
	InvalidUploadGPSLogFilesResponseDataError:                        http.StatusInternalServerError, // ERR000148
	InvalidStatusGetRepNameError:                                     http.StatusInternalServerError, // ERR000150
	GetRepPathError:                                                  http.StatusInternalServerError, // ERR000152
	ConvertGPSLogError:                                               http.StatusInternalServerError, // ERR000154
	GenerateGPXFileContentError:                                      http.StatusInternalServerError, // ERR000155
	WriteGPXFileError:                                                http.StatusInternalServerError, // ERR000156
	NotImplementsError:                                               http.StatusInternalServerError, // ERR000157
	DeleteUsersTagStructError:                                        http.StatusInternalServerError, // ERR000158
	AddUsersTagStructError:                                           http.StatusInternalServerError, // ERR000160
	InvalidUpdateTagStructResponseDataError:                          http.StatusInternalServerError, // ERR000161
	DeleteUsersRepStructError:                                        http.StatusInternalServerError, // ERR000163
	AddUsersRepStructError:                                           http.StatusInternalServerError, // ERR000165
	InvalidUpdateRepStructResponseDataError:                          http.StatusInternalServerError, // ERR000166
	DeleteUsersDeviceStructError:                                     http.StatusInternalServerError, // ERR000168
	AddUsersDeviceStructError:                                        http.StatusInternalServerError, // ERR000170
	InvalidUpdateDeviceStructResponseDataError:                       http.StatusInternalServerError, // ERR000171
	DeleteUsersRepTypeStructError:                                    http.StatusInternalServerError, // ERR000173
	AddUsersRepTypeStructError:                                       http.StatusInternalServerError, // ERR000175
	InvalidUpdateRepTypeStructResponseDataError:                      http.StatusInternalServerError, // ERR000176
	InvalidUpdateAccountStatusResponseDataError:                      http.StatusInternalServerError, // ERR000178
	UpdateUsersAccountStatusError:                                    http.StatusInternalServerError, // ERR000180
	InvalidUpdateUserRepsResponseDataError:                           http.StatusInternalServerError, // ERR000181
	DeleteAllRepositoriesByUserError:                                 http.StatusInternalServerError, // ERR000183
	AccountInvalidAddAccountResponseDataError:                        http.StatusInternalServerError, // ERR000184
	GetAccountError:                                                  http.StatusInternalServerError, // ERR000186
	AddAccountError:                                                  http.StatusInternalServerError, // ERR000188
	InvalidGetGPSLogResponseDataError:                                http.StatusInternalServerError, // ERR000189
	GetGPSLogError:                                                   http.StatusInternalServerError, // ERR000191
	InvalidGetGkillInfoResponseDataError:                             http.StatusInternalServerError, // ERR000192
	InvalidGetShareKyouListInfosResponseDataError:                    http.StatusInternalServerError, // ERR000194
	GetShareKyouListInfosError:                                       http.StatusInternalServerError, // ERR000196
	InvalidDeleteShareKyouListInfosResponseDataError:                 http.StatusInternalServerError, // ERR000197
	DeleteShareKyouListInfosError:                                    http.StatusInternalServerError, // ERR000199
	InvalidGetMiSharedTasksResponseDataError:                         http.StatusInternalServerError, // ERR000200
	GetMiSharedTasksError:                                            http.StatusInternalServerError, // ERR000202
	FindMiKyousError:                                                 http.StatusInternalServerError, // ERR000203
	GetKFTLTemplateError:                                             http.StatusInternalServerError, // ERR000204
	GetTagStructError:                                                http.StatusInternalServerError, // ERR000205
	GetRepStructError:                                                http.StatusInternalServerError, // ERR000206
	GetDeviceStructError:                                             http.StatusInternalServerError, // ERR000207
	GetRepTypeStructError:                                            http.StatusInternalServerError, // ERR000208
	InvalidGetKFTLTemplateResponseDataError:                          http.StatusInternalServerError, // ERR000209
	InvalidAddShareKyouListInfoResponseDataError:                     http.StatusInternalServerError, // ERR000211
	GetShareKyouListInfoError:                                        http.StatusInternalServerError, // ERR000213
	AddShareKyouListInfoError:                                        http.StatusInternalServerError, // ERR000215
	AccountInvalidGenerateTLSFileResponseDataError:                   http.StatusInternalServerError, // ERR000216
	GetTLSFileNamesError:                                             http.StatusInternalServerError, // ERR000218
	GenerateTLSFilesError:                                            http.StatusInternalServerError, // ERR000219
	GetDeviceError:                                                   http.StatusInternalServerError, // ERR000220
	RemoveCertFileError:                                              http.StatusInternalServerError, // ERR000221
	RemovePemFileError:                                               http.StatusInternalServerError, // ERR000222
	GetIDFKyouError:                                                  http.StatusInternalServerError, // ERR000223
	InvalidUpdateApplicationconfigResponseDataError:                  http.StatusInternalServerError, // ERR000224
	UpdateApplicationConfigError:                                     http.StatusInternalServerError, // ERR000226
	InvalidUpdateKFTLTemplateResponseDataError:                       http.StatusInternalServerError, // ERR000227
	DeleteUsersKFTLTemplateError:                                     http.StatusInternalServerError, // ERR000230
	AddUsersKFTLTemplateError:                                        http.StatusInternalServerError, // ERR000231
	InvalidUpdateServerConfigResponseDataError:                       http.StatusInternalServerError, // ERR000232
	UpdateServerConfigError:                                          http.StatusInternalServerError, // ERR000234
	GetAllAccountConfigError:                                         http.StatusInternalServerError, // ERR000235
	GetAllRepositoriesError:                                          http.StatusInternalServerError, // ERR000236
	AddApplicationConfig:                                             http.StatusInternalServerError, // ERR000237
	InvalidGetMiSharedTaskRequest:                                    http.StatusInternalServerError, // ERR000239
	InvalidGetRepositoriesResponseDataError:                          http.StatusInternalServerError, // ERR000240
	GetRepositoriesError:                                             http.StatusInternalServerError, // ERR000242
	GetAllRepNamesError:                                              http.StatusInternalServerError, // ERR000245
	InvalidGetAllRepNamesResponseDataError:                           http.StatusInternalServerError, // ERR000246
	InvalidGetIDFKyouResponseDataError:                               http.StatusInternalServerError, // ERR000248
	AddUpdatedRepositoriesByUser:                                     http.StatusInternalServerError, // ERR000250
	InvalidUpdateIDFKyouResponseDataError:                            http.StatusInternalServerError, // ERR000251
	AddIDFKyouError:                                                  http.StatusInternalServerError, // ERR000253
	FailedMarshalJSONFindQuery:                                       http.StatusInternalServerError, // ERR000255
	InvalidUpdateShareKyouListInfoResponseDataError:                  http.StatusInternalServerError, // ERR000256
	UpdateShareKyouListInfoError:                                     http.StatusInternalServerError, // ERR000258
	FindKyousShareKyouError:                                          http.StatusInternalServerError, // ERR000260
	FindMisShareKyouError:                                            http.StatusInternalServerError, // ERR000261
	FindTagsShareKyouError:                                           http.StatusInternalServerError, // ERR000262
	FindTextsShareKyouError:                                          http.StatusInternalServerError, // ERR000263
	InvalidGetPlayingKyousResponseDataError:                          http.StatusInternalServerError, // ERR000264
	FindKyousPlayingTimeIsError:                                      http.StatusInternalServerError, // ERR000266
	InvalidGetMiTaskNotificationPublicKeyResponseDataError:           http.StatusInternalServerError, // ERR000267
	InvalidRegisterMiTaskNotificationResponse:                        http.StatusInternalServerError, // ERR000269
	GenerateVAPIDKeysError:                                           http.StatusInternalServerError, // ERR000271
	AddGkillNotificationTargetError:                                  http.StatusInternalServerError, // ERR000272
	InvalidAddNotificationResponseDataError:                          http.StatusInternalServerError, // ERR000273
	GetNotificationError:                                             http.StatusInternalServerError, // ERR000275
	AddNotificationError:                                             http.StatusInternalServerError, // ERR000277
	InvalidUpdateNotificationResponseDataError:                       http.StatusInternalServerError, // ERR000278
	UpdateNotificationSuccessMessage:                                 http.StatusInternalServerError, // ERR000281
	InvalidGetNotificationsByTargetIDResponseDataError:               http.StatusInternalServerError, // ERR000282
	GetNotificationsByTargetIDError:                                  http.StatusInternalServerError, // ERR000284
	GetNotificationsByTargetIDSuccessMessage:                         http.StatusInternalServerError, // ERR000285
	InvalidGetNotificationHistoriesByNotificationIDResponseDataError: http.StatusInternalServerError, // ERR000286
	GetNotificationHistoriesByNotificationIDError:                    http.StatusInternalServerError, // ERR000288
	GetNotificationHistoriesByNotificationIDSuccessMessage:           http.StatusInternalServerError, // ERR000289
	GetNotificatorError:                                              http.StatusInternalServerError, // ERR000290
	InvalidRegisterOpenDirectoryResponse:                             http.StatusInternalServerError, // ERR000291
	InvalidRegisterOpenFileResponse:                                  http.StatusInternalServerError, // ERR000293
	OpenFolderError:                                                  http.StatusInternalServerError, // ERR000295
	InvalidReloadRepositoriesResponse:                                http.StatusInternalServerError, // ERR000297
	IDFError:                                                         http.StatusInternalServerError, // ERR000298
	UpdateRepositoryAddressError:                                     http.StatusInternalServerError, // ERR000299
	GetDnoteJSONDataError:                                            http.StatusInternalServerError, // ERR000300
	InvalidUpdateDnoteJSONDataResponseDataError:                      http.StatusInternalServerError, // ERR000301
	DeleteUsersDnoteDataError:                                        http.StatusInternalServerError, // ERR000303
	AddUsersDnoteDataError:                                           http.StatusInternalServerError, // ERR000304
	AddKCError:                                                       http.StatusInternalServerError, // ERR000305
	GetKCError:                                                       http.StatusInternalServerError, // ERR000306
	InvalidUpdateKCResponseDataError:                                 http.StatusInternalServerError, // ERR000309
	AccountInvalidAddKCResponseDataError:                             http.StatusInternalServerError, // ERR000311
	InvalidGetKCResponseDataError:                                    http.StatusInternalServerError, // ERR000313
	InvalidGetUpdatedDatasByTimeResponse:                             http.StatusInternalServerError, // ERR000315
	GetLatestDataRepositoryAddressByUpdateTimeAfterError:             http.StatusInternalServerError, // ERR000317
	AccountInvalidCommitTxResponseDataError:                          http.StatusInternalServerError, // ERR000318
	CommitTxGetKmemoError:                                            http.StatusInternalServerError, // ERR000320
	CommitTxGetKCError:                                               http.StatusInternalServerError, // ERR000321
	CommitTxGetIDFKyouError:                                          http.StatusInternalServerError, // ERR000322
	CommitTxGetLantanaError:                                          http.StatusInternalServerError, // ERR000323
	CommitTxGetMiError:                                               http.StatusInternalServerError, // ERR000324
	CommitTxGetNlogError:                                             http.StatusInternalServerError, // ERR000325
	CommitTxGetNotificationError:                                     http.StatusInternalServerError, // ERR000326
	CommitTxGetReKyouError:                                           http.StatusInternalServerError, // ERR000327
	CommitTxGetTagError:                                              http.StatusInternalServerError, // ERR000328
	CommitTxGetTextError:                                             http.StatusInternalServerError, // ERR000329
	CommitTxGetTimeIsError:                                           http.StatusInternalServerError, // ERR000330
	CommitTxGetURLogError:                                            http.StatusInternalServerError, // ERR000331
	AccountInvalidDiscardTxResponseDataError:                         http.StatusInternalServerError, // ERR000332
	CommitTxDeleteIDFKyouError:                                       http.StatusInternalServerError, // ERR000334
	CommitTxDeleteKCError:                                            http.StatusInternalServerError, // ERR000335
	CommitTxDeleteKmemoError:                                         http.StatusInternalServerError, // ERR000336
	CommitTxDeleteLantanaError:                                       http.StatusInternalServerError, // ERR000337
	CommitTxDeleteMiError:                                            http.StatusInternalServerError, // ERR000338
	CommitTxDeleteNlogError:                                          http.StatusInternalServerError, // ERR000339
	CommitTxDeleteNotificationError:                                  http.StatusInternalServerError, // ERR000340
	CommitTxDeleteReKyouError:                                        http.StatusInternalServerError, // ERR000341
	CommitTxDeleteTagError:                                           http.StatusInternalServerError, // ERR000342
	CommitTxDeleteTextError:                                          http.StatusInternalServerError, // ERR000343
	CommitTxDeleteTimeIsError:                                        http.StatusInternalServerError, // ERR000344
	CommitTxDeleteURLogError:                                         http.StatusInternalServerError, // ERR000345
	NotFoundTLSCertFileError:                                         http.StatusInternalServerError, // ERR000346
	NotFoundTLSKeyFileError:                                          http.StatusInternalServerError, // ERR000347
	GetAccountSessionsError:                                          http.StatusInternalServerError, // ERR000348
	AddURLogLoginSessionError:                                        http.StatusInternalServerError, // ERR000349
	SubmitKFTLTextError:                                              http.StatusInternalServerError, // ERR000351
	InvalidGetKyousMCPResponseDataError:                              http.StatusInternalServerError, // ERR000353
	GetKyousMCPError:                                                 http.StatusInternalServerError, // ERR000354
	InvalidUpdateCacheResponseDataError:                              http.StatusInternalServerError, // ERR000356
	UpdateCacheError:                                                 http.StatusInternalServerError, // ERR000357
	InvalidUpdateMiResponseDataError:                                 http.StatusInternalServerError, // ERR000358
	InvalidGetKyouResponseDataError:                                  http.StatusInternalServerError, // ERR000359
	UpdateTagError:                                                   http.StatusInternalServerError, // ERR000360
	UpdateTextError:                                                  http.StatusInternalServerError, // ERR000361
	UpdateKmemoError:                                                 http.StatusInternalServerError, // ERR000362
	UpdateURLogError:                                                 http.StatusInternalServerError, // ERR000363
	UpdateNlogError:                                                  http.StatusInternalServerError, // ERR000364
	UpdateTimeIsError:                                                http.StatusInternalServerError, // ERR000365
	UpdateLantanaError:                                               http.StatusInternalServerError, // ERR000366
	UpdateMiError:                                                    http.StatusInternalServerError, // ERR000367
	UpdateReKyouError:                                                http.StatusInternalServerError, // ERR000368
	UpdateIDFKyouError:                                               http.StatusInternalServerError, // ERR000369
	UpdateKCError:                                                    http.StatusInternalServerError, // ERR000370
	UpdateNotificationError:                                          http.StatusInternalServerError, // ERR000371
	BrowseZipContentsError:                                           http.StatusInternalServerError, // ERR000376
	GetPluginListError:                                               http.StatusInternalServerError, // ERR000378
	GetPluginContentHTMLError:                                        http.StatusInternalServerError, // ERR000380
	GetPluginConfigHTMLError:                                         http.StatusInternalServerError, // ERR000382
	PostPluginConfigError:                                            http.StatusInternalServerError, // ERR000384
	GetIDFKyouByRelativePathError:                                    http.StatusInternalServerError, // ERR000386
	InvalidAddMiReKyouResponseDataError:                              http.StatusInternalServerError, // ERR000391
	AddMiReKyouError:                                                 http.StatusInternalServerError, // ERR000392
	InvalidUpdateMiReKyouResponseDataError:                           http.StatusInternalServerError, // ERR000395
	UpdateMiReKyouError:                                              http.StatusInternalServerError, // ERR000396
	InvalidGetMiReKyouResponseDataError:                              http.StatusInternalServerError, // ERR000399
	GetMiReKyouError:                                                 http.StatusInternalServerError, // ERR000400
	CommitTxGetMiReKyouError:                                         http.StatusInternalServerError, // ERR000401
	InvalidGetReKyousByTargetIDResponseDataError:                     http.StatusInternalServerError, // ERR000403
	GetReKyousByTargetIDError:                                        http.StatusInternalServerError, // ERR000404
	InvalidGetMiReKyousByTargetIDResponseDataError:                   http.StatusInternalServerError, // ERR000406
	GetMiReKyousByTargetIDError:                                      http.StatusInternalServerError, // ERR000407
	FindKyousError:                                                   http.StatusInternalServerError, // ERR000410
	InvalidGetRepInfosMCPResponseDataError:                           http.StatusInternalServerError, // ERR000412
	InternalServerPanicError:                                         http.StatusInternalServerError, // ERR000415
	ReadRequestBodyError:                                             http.StatusInternalServerError, // ERR000418
}
