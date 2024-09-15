package api

type TagFilterMode string

const (
	Or TagFilterMode = "or"

	And TagFilterMode = "and"
)
