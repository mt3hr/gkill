package message

// TODO モデル反映
const (
	AccountInvalidLoginRequestDataError                = "ERR000001"
	AccountNotFoundError                               = "ERR000002"
	AccountIsNotEnableError                            = "ERR000003"
	AccountPasswordResetTokenIsNotNilError             = "ERR000004"
	AccountInvalidPasswordError                        = "ERR000005"
	AccountLoginInternalServerError                    = "ERR000006"
	AccountInvalidLoginResponseDataError               = "ERR000007"
	AccountInvalidLogoutRequestDataError               = "ERR000008"
	AccountInvalidLogoutResponseDataError              = "ERR000009"
	AccountLogoutInternalServerError                   = "ERR000010"
	AccountInvalidResetPasswordResponseDataError       = "ERR000011"
	AccountInvalidResetPasswordRequestDataError        = "ERR000012"
	AccountSessionNotFoundError                        = "ERR000013"
	AccountNotHasAdminError                            = "ERR000014"
	AccountInfoUpdateError                             = "ERR000015"
	AccountInvalidSetNewPasswordRequestDataError       = "ERR000016"
	AccountInvalidSetNewPasswordResponseDataError      = "ERR000017"
	RepositoriesGetError                               = "ERR000018"
	AddTagError                                        = "ERR000019"
	GetTagError                                        = "ERR000020"
	AddTextError                                       = "ERR000021"
	GetTextError                                       = "ERR000022"
	AddKmemoError                                      = "ERR000023"
	GetKmemoError                                      = "ERR000024"
	InvalidAddTagRequestDataError                      = "ERR000025"
	InvalidAddTagResponseDataError                     = "ERR000026"
	InvalidAddTextRequestDataError                     = "ERR000027"
	InvalidAddTextResponseDataError                    = "ERR000028"
	AddURLogError                                      = "ERR000029"
	GetURLogError                                      = "ERR000030"
	InvalidAddURLogRequestDataError                    = "ERR000031"
	InvalidAddURLogResponseDataError                   = "ERR000032"
	AddNlogError                                       = "ERR000033"
	GetNlogError                                       = "ERR000034"
	InvalidAddNlogRequestDataError                     = "ERR000035"
	InvalidAddNlogResponseDataError                    = "ERR000036"
	AddTimeIsError                                     = "ERR000037"
	GetTimeIsError                                     = "ERR000038"
	InvalidAddTimeIsRequestDataError                   = "ERR000039"
	InvalidAddTimeIsResponseDataError                  = "ERR000040"
	AddLantanaError                                    = "ERR000041"
	GetLantanaError                                    = "ERR000042"
	InvalidAddLantanaRequestDataError                  = "ERR000043"
	InvalidAddLantanaResponseDataError                 = "ERR000044"
	AddKyouInfoError                                   = "ERR000045"
	GetKyouInfoError                                   = "ERR000046"
	InvalidAddKyouInfoRequestDataError                 = "ERR000047"
	InvalidAddKyouInfoResponseDataError                = "ERR000048"
	AddReKyouError                                     = "ERR000049"
	GetReKyouError                                     = "ERR000050"
	InvalidAddReKyouRequestDataError                   = "ERR000051"
	InvalidAddReKyouResponseDataError                  = "ERR000052"
	InvalidUpdateTagRequestDataError                   = "ERR000053"
	InvalidUpdateTagResponseDataError                  = "ERR000054"
	NotFoundTagError                                   = "ERR000055"
	AleadyExistTagError                                = "ERR000056"
	AleadyExistTextError                               = "ERR000057"
	AleadyExistKmemoError                              = "ERR000058"
	AleadyExistURLogError                              = "ERR000059"
	AleadyExistNlogError                               = "ERR000060"
	AleadyExistTimeIsError                             = "ERR000061"
	AleadyExistLantanaError                            = "ERR000062"
	AleadyExistKyouInfoError                           = "ERR000063"
	AleadyExistReKyouError                             = "ERR000064"
	InvalidUpdateTextRequestDataError                  = "ERR000065"
	InvalidUpdateTextResponseDataError                 = "ERR000066"
	NotFoundTextError                                  = "ERR000067"
	InvalidUpdateKmemoRequestDataError                 = "ERR000068"
	InvalidUpdateKmemoResponseDataError                = "ERR000069"
	NotFoundKmemoError                                 = "ERR000070"
	InvalidUpdateURLogRequestDataError                 = "ERR000071"
	InvalidUpdateURLogResponseDataError                = "ERR000072"
	NotFoundURLogError                                 = "ERR000073"
	InvalidUpdateNlogRequestDataError                  = "ERR000074"
	InvalidUpdateNlogResponseDataError                 = "ERR000075"
	NotFoundNlogError                                  = "ERR000076"
	InvalidUpdateTimeIsRequestDataError                = "ERR000077"
	InvalidUpdateTimeIsResponseDataError               = "ERR000078"
	NotFoundTimeIsError                                = "ERR000079"
	InvalidUpdateLantanaRequestDataError               = "ERR000080"
	InvalidUpdateLantanaResponseDataError              = "ERR000081"
	NotFoundLantanaError                               = "ERR000082"
	AddMiError                                         = "ERR000083"
	GetMiError                                         = "ERR000084"
	InvalidAddMiRequestDataError                       = "ERR000085"
	InvalidAddMiResponseDataError                      = "ERR000086"
	AleadyExistMiError                                 = "ERR000087"
	InvalidUpdateMiRequestDataError                    = "ERR000088"
	InvalidUpdateMiResponseDataError                   = "ERR000088"
	NotFoundMiError                                    = "ERR000089"
	InvalidUpdateKyouInfoRequestDataError              = "ERR000090"
	InvalidUpdateKyouInfoResponseDataError             = "ERR000091"
	NotFoundKyouInfoError                              = "ERR000092"
	InvalidUpdateReKyouRequestDataError                = "ERR000093"
	InvalidUpdateReKyouResponseDataError               = "ERR000094"
	NotFoundReKyouError                                = "ERR000095"
	AccountInvalidAddKmemoResponseDataError            = "ERR000096"
	AccountInvalidAddKmemoRequestDataError             = "ERR000097"
	InvalidGetKyousResponseDataError                   = "ERR000098"
	InvalidGetKyousRequestDataError                    = "ERR000099"
	InvalidGetKyouResponseDataError                    = "ERR000099"
	InvalidGetKyouRequestDataError                     = "ERR000100"
	GetKyouError                                       = "ERR000101"
	InvalidGetKmemoResponseDataError                   = "ERR000102"
	InvalidGetKmemoRequestDataError                    = "ERR000103"
	InvalidGetURLogResponseDataError                   = "ERR000104"
	InvalidGetURLogRequestDataError                    = "ERR000105"
	InvalidGetNlogResponseDataError                    = "ERR000106"
	InvalidGetNlogRequestDataError                     = "ERR000107"
	InvalidGetTimeIsResponseDataError                  = "ERR000108"
	InvalidGetTimeIsRequestDataError                   = "ERR000109"
	InvalidGetMiResponseDataError                      = "ERR000110"
	InvalidGetMiRequestDataError                       = "ERR000111"
	InvalidGetLantanaResponseDataError                 = "ERR000112"
	InvalidGetLantanaRequestDataError                  = "ERR000113"
	InvalidGetReKyouResponseDataError                  = "ERR000114"
	InvalidGetReKyouRequestDataError                   = "ERR000115"
	InvalidGetGitCommitLogResponseDataError            = "ERR000116"
	InvalidGetGitCommitLogRequestDataError             = "ERR000117"
	GetGitCommitLogError                               = "ERR000118"
	InvalidGetGitCommitLogsResponseDataError           = "ERR000119"
	InvalidGetGitCommitLogsRequestDataError            = "ERR000120"
	GetGitCommitLogsError                              = "ERR000121"
	InvalidGetMiBoardNamesResponseDataError            = "ERR000122"
	InvalidGetMiBoardNamesRequestDataError             = "ERR000123"
	GetMiBoardNamesError                               = "ERR000124"
	InvalidGetAllTagNamesResponseDataError             = "ERR000125"
	InvalidGetAllTagNamesRequestDataError              = "ERR000126"
	GetAllTagNamesError                                = "ERR000127"
	InvalidGetTagsByTargetIDResponseDataError          = "ERR000128"
	InvalidGetTagsByTargetIDRequestDataError           = "ERR000129"
	GetTagsByTargetIDError                             = "ERR000130"
	InvalidGetTagHistoriesByTagIDResponseDataError     = "ERR000131"
	InvalidGetTagHistoriesByTagIDRequestDataError      = "ERR000132"
	GetTagHistoriesByTagIDError                        = "ERR000133"
	InvalidGetTextHistoriesByTextIDResponseDataError   = "ERR000134"
	InvalidGetTextHistoriesByTextIDRequestDataError    = "ERR000135"
	GetTextHistoriesByTextIDError                      = "ERR000136"
	InvalidGetTextsByTargetIDResponseDataError         = "ERR000137"
	InvalidGetTextsByTargetIDRequestDataError          = "ERR000138"
	GetTextsByTargetIDError                            = "ERR000139"
	InvalidGetApplicationConfigResponseDataError       = "ERR000140"
	InvalidGetApplicationConfigRequestDataError        = "ERR000141"
	GetApplicationConfigError                          = "ERR000142"
	InvalidGetServerConfigResponseDataError            = "ERR000143"
	InvalidGetServerConfigRequestDataError             = "ERR000144"
	GetServerConfigError                               = "ERR000145"
	InvalidUploadFilesResponseDataError                = "ERR000146"
	InvalidUploadFilesRequestDataError                 = "ERR000147"
	InvalidUploadGPSLogFilesResponseDataError          = "ERR000148"
	InvalidUploadGPSLogFilesRequestDataError           = "ERR000149"
	InvalidStatusGetRepNameError                       = "ERR000150"
	NotFoundTargetIDFRepError                          = "ERR000151"
	GetRepPathError                                    = "ERR000152"
	NotFoundTargetGPSLogRepError                       = "ERR000153"
	ConvertGPSLogError                                 = "ERR000154"
	GenerateGPXFileContentError                        = "ERR000155"
	WriteGPXFileError                                  = "ERR000156"
	NotImplementsError                                 = "ERR000157"
	DeleteUsersTagStructError                          = "ERR000158"
	TagStructInvalidUserID                             = "ERR000159"
	AddUsersTagStructError                             = "ERR000160"
	InvalidUpdateTagStructResponseDataError            = "ERR000161"
	InvalidUpdateTagStructRequestDataError             = "ERR000162"
	DeleteUsersRepStructError                          = "ERR000163"
	RepStructInvalidUserID                             = "ERR000164"
	AddUsersRepStructError                             = "ERR000165"
	InvalidUpdateRepStructResponseDataError            = "ERR000166"
	InvalidUpdateRepStructRequestDataError             = "ERR000167"
	DeleteUsersDeviceStructError                       = "ERR000168"
	DeviceStructInvalidUserID                          = "ERR000169"
	AddUsersDeviceStructError                          = "ERR000170"
	InvalidUpdateDeviceStructResponseDataError         = "ERR000171"
	InvalidUpdateDeviceStructRequestDataError          = "ERR000172"
	DeleteUsersRepTypeStructError                      = "ERR000173"
	RepTypeStructInvalidUserID                         = "ERR000174"
	AddUsersRepTypeStructError                         = "ERR000175"
	InvalidUpdateRepTypeStructResponseDataError        = "ERR000176"
	InvalidUpdateRepTypeStructRequestDataError         = "ERR000177"
	InvalidUpdateAccountStatusResponseDataError        = "ERR000178"
	InvalidUpdateAccountStatusRequestDataError         = "ERR000179"
	UpdateUsersAccountStatusError                      = "ERR000180"
	InvalidUpdateUserRepsResponseDataError             = "ERR000181"
	InvalidUpdateUserRepsRequestDataError              = "ERR000182"
	DeleteAllRepositoriesByUserError                   = "ERR000183"
	AccountInvalidAddAccountResponseDataError          = "ERR000184"
	AccountInvalidAddAccountRequestDataError           = "ERR000185"
	GetAccountError                                    = "ERR000186"
	AleadyExistAccountError                            = "ERR000187"
	AddAccountError                                    = "ERR000188"
	InvalidGetGPSLogResponseDataError                  = "ERR000189"
	InvalidGetGPSLogRequestDataError                   = "ERR000190"
	GetGPSLogError                                     = "ERR000191"
	InvalidGetGkillInfoResponseDataError               = "ERR000192"
	InvalidGetGkillInfoRequestDataError                = "ERR000193"
	InvalidGetShareMiTaskListInfosResponseDataError    = "ERR000194"
	InvalidGetShareMiTaskListInfosRequestDataError     = "ERR000195"
	GetShareMiTaskListInfosError                       = "ERR000196"
	InvalidDeleteShareMiTaskListInfosResponseDataError = "ERR000197"
	InvalidDeleteShareMiTaskListInfosRequestDataError  = "ERR000198"
	DeleteShareMiTaskListInfosError                    = "ERR000199"
	InvalidGetMiSharedTasksResponseDataError           = "ERR000200"
	InvalidGetMiSharedTasksRequestDataError            = "ERR000201"
	GetMiSharedTasksError                              = "ERR000202"
	FindMiKyousError                                   = "ERR000203"
	GetKFTLTemplateError                               = "ERR000204"
	GetTagStructError                                  = "ERR000205"
	GetRepStructError                                  = "ERR000206"
	GetDeviceStructError                               = "ERR000207"
	GetRepTypeStructError                              = "ERR000208"
	InvalidGetKFTLTemplateResponseDataError            = "ERR000209"
	InvalidGetKFTLTemplateRequestDataError             = "ERR000210"
	InvalidAddShareMiTaskListInfoResponseDataError     = "ERR000211"
	InvalidAddShareMiTaskListInfoRequestDataError      = "ERR000212"
	GetShareMiTaskListInfoError                        = "ERR000213"
	AleadyExistShareMiTaskListInfoError                = "ERR000214"
	AddShareMiTaskListInfoError                        = "ERR000215"
	AccountInvalidGenerateTLSFileResponseDataError     = "ERR000216"
	AccountInvalidGenerateTLSFileRequestDataError      = "ERR000217"
	GetTLSFileNamesError                               = "ERR000218"
	GenerateTLSFilesError                              = "ERR000219"
	GetDeviceError                                     = "ERR000220"
	RemoveCertFileError                                = "ERR000221"
	RemovePemFileError                                 = "ERR000222"
)
