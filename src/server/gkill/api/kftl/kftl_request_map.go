package kftl

import (
	"fmt"
	"slices"
)

// KFTLRequestMap maps requestID → KFTLRequest, preserving insertion order.
// If an existing entry is a KFTLPrototypeRequest, the new entry inherits its
// tags, texts, and related_time (prototype inheritance).
// Mirrors: src/classes/kftl/kftl-request-map.ts
type KFTLRequestMap struct {
	entries map[string]KFTLRequest
	order   []string // preserves insertion order
}

// NewKFTLRequestMap creates an empty KFTLRequestMap.
func NewKFTLRequestMap() *KFTLRequestMap {
	return &KFTLRequestMap{
		entries: make(map[string]KFTLRequest),
	}
}

// Set sets a request in the map.
// If an entry already exists and is a KFTLPrototypeRequest, the new request
// inherits tags/texts/related_time from it.
// If an entry already exists and is NOT a prototype, returns an error.
func (m *KFTLRequestMap) Set(requestID string, req KFTLRequest) error {
	existing, ok := m.entries[requestID]
	if ok {
		proto, isProto := existing.(*KFTLPrototypeRequest)
		if !isProto {
			return newKFTLInputError("KFTL_REQUEST_ALREADY_SET_ERROR_MESSAGE",
				fmt.Errorf("request id=%s is already set and is not a prototype", requestID))
		}
		// Inherit from prototype
		for _, tag := range proto.GetTags() {
			req.AddTag(tag)
		}
		for textID, textContent := range proto.GetTextsMap() {
			if textContent != "" {
				req.AddTextLine(textID, textContent)
			}
		}
		if rt := proto.relatedTime; rt != nil {
			req.SetRelatedTimePtr(rt)
		}
	} else {
		m.order = append(m.order, requestID)
	}
	m.entries[requestID] = req
	return nil
}

// Get returns the request for the given ID, and whether it was found.
func (m *KFTLRequestMap) Get(requestID string) (KFTLRequest, bool) {
	req, ok := m.entries[requestID]
	return req, ok
}

// Delete は役目を終えたプロトタイプを外す（順序からも消す）。
//
// 使うのは `？時刻` の直後の `ーん` だけ —— 関連時刻をブロックへ取り込んだあとも
// プロトタイプが map に残ると、validateRequestContents が「付け先の無い関連時刻」として弾く。
// 他のプロトタイプは次の記録の Set が置き換えるので、消す場面は無い。
func (m *KFTLRequestMap) Delete(requestID string) {
	if _, ok := m.entries[requestID]; !ok {
		return
	}
	delete(m.entries, requestID)
	m.order = slices.DeleteFunc(m.order, func(id string) bool { return id == requestID })
}

// ReplaceAll は展開後のリクエスト列でマップを置き換える。
// 繰り返しの展開でだけ使う（元のリクエストは複製に置き換わるので、消してから入れ直す）。
func (m *KFTLRequestMap) ReplaceAll(reqs []KFTLRequest) {
	m.entries = make(map[string]KFTLRequest, len(reqs))
	m.order = m.order[:0]
	for _, req := range reqs {
		id := req.GetRequestID()
		if _, ok := m.entries[id]; !ok {
			m.order = append(m.order, id)
		}
		m.entries[id] = req
	}
}

// All returns all requests in insertion order.
func (m *KFTLRequestMap) All() []KFTLRequest {
	result := make([]KFTLRequest, 0, len(m.order))
	for _, id := range m.order {
		if req, ok := m.entries[id]; ok {
			result = append(result, req)
		}
	}
	return result
}
