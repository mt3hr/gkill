'use strict'

import type { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'

// TagStruct（タグ構成）のツリーに指定されたタグ名が存在するかを再帰的に調べる
export function tag_exists_in_tag_struct(tag_name: string, struct: TagStructElementData): boolean {
    if (struct.tag_name === tag_name) return true
    if (struct.children) {
        for (const child of struct.children) {
            if (tag_exists_in_tag_struct(tag_name, child)) return true
        }
    }
    return false
}
