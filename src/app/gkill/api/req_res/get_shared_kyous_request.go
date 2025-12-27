package req_res

type GetSharedKyousRequest struct {
	SharedID string `json:"shared_id"`

	LocaleName string `json:"locale_name"`
}
