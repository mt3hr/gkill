const dnote_key_getter_dictionary = new Map<string, { from_json: (json: Record<string, unknown>) => unknown }>()
export default dnote_key_getter_dictionary
// 循環参照対策で各KeyGetterの.tsファイルから登録