const dnote_kyou_filter_dictionary = new Map<string, { from_json: (json: Record<string, unknown>) => unknown }>()
export default dnote_kyou_filter_dictionary
// 循環参照対策で各KeyGetterの.tsファイルから登録