'use strict'

import type { TagStructElementData } from "@/classes/datas/config/tag-struct-element-data"

// タグツリーから「初期チェックあり」のタグ名を集める。
// FindKyouQuery の既定クエリ生成と、TimeIsタグツリーの null フォールバック
// （クエリ上 timeis_tags=null のときにツリーへ出す初期チェック集合）が共用する。
export function collect_inited_tag_names(tag_struct: TagStructElementData): Array<string> {
    const tag_names = new Array<string>()
    const walk = (tag: TagStructElementData): void => {
        const tag_children = tag.children
        if (tag_children) {
            tag_children.forEach(child_tag => {
                if (child_tag.check_when_inited) {
                    tag_names.push(child_tag.tag_name)
                }
                if (child_tag) {
                    walk(child_tag)
                }
            })
        }
    }
    walk(tag_struct)
    return tag_names
}
