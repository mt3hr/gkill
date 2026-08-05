const predicate_dictonary = new Map<string, { from_json: (json: Record<string, unknown>) => unknown }>()
export default predicate_dictonary
// 循環参照対策で各Predicateの.tsファイルから登録