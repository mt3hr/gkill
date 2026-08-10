'use strict'

import { GkillError } from '@/classes/api/gkill-error'
import { i18n } from '@/i18n'
import { TagStructElementData } from './tag-struct-element-data'
import { RepStructElementData } from './rep-struct-element-data'
import { DeviceStructElementData } from './device-struct-element-data'
import { RepTypeStructElementData } from './rep-type-struct-element-data'
import { generate_rep_type_map } from './rep-type-map'
import { KFTLTemplateElementData } from '../kftl-template-element-data'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetAllRepNamesRequest } from '@/classes/api/req_res/get-all-rep-names-request'
import { GetAllTagNamesRequest } from '@/classes/api/req_res/get-all-tag-names-request'
import { MiBoardStructElementData } from './mi-board-struct-element-data'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import type { FoldableStructModel } from '@/pages/views/foldable-struct-model'

/**
 * ツリー全体を探して条件に合うノードがあるか調べる。
 *
 * 「無ければ足す」系(`append_no_tags` / `append_no_devices` / `append_all_mi_board`)は
 * 直下の children しか見ておらず、ユーザーがそのノードをフォルダへ移動すると
 * 読み込みのたびにルート直下へ再生成されて重複していた。
 * mi_board_struct には編集UIが無かったので到達不能だったが、
 * 板構造の編集UIを足した時点で踏めるようになる
 */
function has_struct_node(root: FoldableStructModel | null | undefined, matches: (node: FoldableStructModel) => boolean): boolean {
    if (!root) {
        return false
    }
    if (matches(root)) {
        return true
    }
    for (const child of root.children ?? []) {
        if (has_struct_node(child, matches)) {
            return true
        }
    }
    return false
}

export class ApplicationConfig {
    is_loaded: boolean
    user_id: string
    device: string
    use_dark_theme: boolean
    google_map_api_key: string
    rykv_image_list_column_number: number
    rykv_hot_reload: boolean
    mi_default_board: string
    rykv_default_period: number
    mi_default_period: number
    account_is_admin: boolean
    dnote_json_data: unknown
    ryuu_json_data: unknown
    dashboard_json_data: unknown
    plaing_timeis_json_data: unknown
    saved_find_query_json_data: unknown

    user_is_admin: boolean
    cache_clear_count_limit: number
    global_ip: string
    private_ip: string
    lan_hostname: string
    global_hostname: string
    version: string
    build_time: Date
    commit_hash: string

    is_show_share_footer: boolean
    default_page: string
    kftl_template_struct: KFTLTemplateElementData
    tag_struct: TagStructElementData
    rep_struct: RepStructElementData
    device_struct: DeviceStructElementData
    rep_type_struct: RepTypeStructElementData
    mi_board_struct: MiBoardStructElementData
    show_tags_in_list: boolean
    show_tutorial_on_startup!: boolean
    session_is_local: boolean
    urlog_bookmarklet_session: string

    for_share_kyou: boolean

    clone(): ApplicationConfig {
        const application_config = new ApplicationConfig()
        application_config.is_loaded = this.is_loaded
        application_config.user_id = this.user_id
        application_config.device = this.device
        application_config.use_dark_theme = this.use_dark_theme
        application_config.google_map_api_key = this.google_map_api_key
        application_config.rykv_image_list_column_number = this.rykv_image_list_column_number
        application_config.rykv_hot_reload = this.rykv_hot_reload
        application_config.mi_default_board = this.mi_default_board
        application_config.tag_struct = (JSON.parse(JSON.stringify(this.tag_struct)) as TagStructElementData)
        application_config.rep_struct = (JSON.parse(JSON.stringify(this.rep_struct)) as RepStructElementData)
        application_config.device_struct = (JSON.parse(JSON.stringify(this.device_struct)) as DeviceStructElementData)
        application_config.rep_type_struct = (JSON.parse(JSON.stringify(this.rep_type_struct)) as RepTypeStructElementData)
        // 編集ダイアログは cloned_application_config の構造を in-place で書き換える
        // (children.push / splice / move_struct_*)。参照コピーのままだと
        // 実体まで到達してしまい、キャンセルしても編集が取り消せない
        application_config.kftl_template_struct = (JSON.parse(JSON.stringify(this.kftl_template_struct)) as KFTLTemplateElementData)
        application_config.mi_board_struct = (JSON.parse(JSON.stringify(this.mi_board_struct)) as MiBoardStructElementData)
        application_config.dnote_json_data = this.dnote_json_data
        application_config.ryuu_json_data = this.ryuu_json_data
        application_config.dashboard_json_data = this.dashboard_json_data
        // JSONデータは保存時に丸ごと差し替える運用なので参照コピーでよい（dashboard等と同じ）
        application_config.plaing_timeis_json_data = this.plaing_timeis_json_data
        application_config.saved_find_query_json_data = this.saved_find_query_json_data
        application_config.account_is_admin = this.account_is_admin
        application_config.session_is_local = this.session_is_local
        application_config.urlog_bookmarklet_session = this.urlog_bookmarklet_session
        application_config.for_share_kyou = this.for_share_kyou
        application_config.rykv_default_period = this.rykv_default_period
        application_config.mi_default_period = this.mi_default_period
        application_config.is_show_share_footer = this.is_show_share_footer
        application_config.show_tags_in_list = this.show_tags_in_list
        application_config.show_tutorial_on_startup = this.show_tutorial_on_startup
        application_config.default_page = this.default_page

        application_config.user_id = this.user_id
        application_config.device = this.device
        application_config.user_is_admin = this.user_is_admin
        application_config.cache_clear_count_limit = this.cache_clear_count_limit
        application_config.global_ip = this.global_ip
        application_config.private_ip = this.private_ip
        application_config.lan_hostname = this.lan_hostname
        application_config.global_hostname = this.global_hostname
        application_config.version = this.version
        application_config.build_time = this.build_time
        application_config.commit_hash = this.commit_hash
        return application_config
    }
    async append_not_found_infos(): Promise<Array<GkillError>> {
        const errors = new Array<GkillError>()
        // rep名は下の3つで共通なので、ここで1回だけ取って渡す
        const rep_names_res = await resolve_rep_names()
        if (rep_names_res.errors.length !== 0) {
            return rep_names_res.errors
        }
        const rep_names = rep_names_res.rep_names
        errors.push(...await this.append_not_found_reps(rep_names))
        errors.push(...await this.append_not_found_devices(rep_names))
        errors.push(...await this.append_not_found_rep_types(rep_names))
        errors.push(...await this.append_not_found_tags())
        errors.push(...await this.append_not_found_mi_boards())
        errors.push(...await this.append_no_devices())
        errors.push(...await this.append_no_tags())
        errors.push(...await this.append_all_mi_board())
        return errors
    }
    async load_all(): Promise<Array<GkillError>> {
        const errors = Array<GkillError>()
        errors.push(...await this.append_not_found_infos())
        return errors
    }
    async append_not_found_reps(cached_rep_names?: Array<string>): Promise<Array<GkillError>> {
        const res = await resolve_rep_names(cached_rep_names)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        const not_found = new Set<string>()

        res.rep_names.forEach(rep_name => {
            let exist = false
            let rep_name_walk = (_rep: RepStructElementData): void => { }
            rep_name_walk = (rep: RepStructElementData): void => {
                const rep_children = rep.children
                if (rep_name === rep.rep_name) {
                    exist = true
                }
                if (rep_children) {
                    rep_children.forEach(child_rep => {
                        if (child_rep) {
                            rep_name_walk(child_rep)
                        }
                    })
                }
            }
            rep_name_walk(this.rep_struct)
            if (!exist) {
                not_found.add(rep_name)
            }
        })

        not_found.forEach(rep_name => {
            const rep_struct = new RepStructElementData()
            rep_struct.key = rep_name
            rep_struct.name = rep_name
            rep_struct.check_when_inited = true
            rep_struct.is_checked = rep_struct.check_when_inited
            rep_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            rep_struct.ignore_check_rep_rykv = false
            rep_struct.rep_name = rep_name
            rep_struct.is_dir = false
            if (!this.rep_struct.children) {
                this.rep_struct.children = []
            }
            this.rep_struct.children.push(rep_struct)
        })
        return new Array<GkillError>()
    }
    async append_not_found_tags(): Promise<Array<GkillError>> {
        const collect_not_found = (tag_names: Array<string>): Set<string> => {
            const not_found = new Set<string>()
            tag_names.forEach(tag_name => {
                let exist = false
                let tag_name_walk = (_tag: TagStructElementData): void => { }
                tag_name_walk = (tag: TagStructElementData): void => {
                    const tag_children = tag.children
                    if (tag_name === tag.tag_name) {
                        exist = true
                    }
                    if (tag_children) {
                        tag_children.forEach(child_tag => {
                            if (child_tag) {
                                tag_name_walk(child_tag)
                            }
                        })
                    }
                }
                tag_name_walk(this.tag_struct)
                if (!exist) {
                    not_found.add(tag_name)
                }
            })
            return not_found
        }

        const req = new GetAllTagNamesRequest()

        const res = await GkillAPI.get_gkill_api().get_all_tag_names(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }

        let not_found = collect_not_found(res.tag_names)

        // TagStructは永続化されるので、ServiceWorkerがキャッシュした古い一覧に含まれる
        // 編集前のタグ名を書き込まないよう、追加するものがあるときだけサーバに問い直す
        if (not_found.size !== 0) {
            const reget_req = new GetAllTagNamesRequest()
            reget_req.force_reget = true
            const reget_res = await GkillAPI.get_gkill_api().get_all_tag_names(reget_req)
            if (reget_res.errors && reget_res.errors.length !== 0) {
                return reget_res.errors
            }
            not_found = collect_not_found(reget_res.tag_names)
        }

        not_found.forEach(tag_name => {
            const tag_struct = new TagStructElementData()
            tag_struct.key = tag_name
            tag_struct.name = tag_name
            tag_struct.check_when_inited = true
            tag_struct.is_checked = tag_struct.check_when_inited
            tag_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            tag_struct.is_force_hide = false
            tag_struct.tag_name = tag_name
            tag_struct.is_dir = false
            if (!this.tag_struct.children) {
                this.tag_struct.children = []
            }
            this.tag_struct.children.push(tag_struct)
        })
        return new Array<GkillError>()
    }

    async append_not_found_mi_boards(): Promise<Array<GkillError>> {
        const collect_not_found = (board_names: Array<string>): Set<string> => {
            const not_found = new Set<string>()
            board_names.forEach(board_name => {
                let exist = false
                let board_name_walk = (_board_name: MiBoardStructElementData): void => { }
                board_name_walk = (board: MiBoardStructElementData): void => {
                    const board_children = board.children
                    if (board_name === board.board_name) {
                        exist = true
                    }
                    if (board_children) {
                        board_children.forEach(child_board => {
                            if (child_board) {
                                board_name_walk(child_board)
                            }
                        })
                    }
                }
                board_name_walk(this.mi_board_struct)
                if (!exist) {
                    not_found.add(board_name)
                }
            })
            return not_found
        }

        const req = new GetMiBoardRequest()

        const res = await GkillAPI.get_gkill_api().get_mi_board_list(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }

        let not_found = collect_not_found(res.boards)

        // MiBoardStructは永続化されるので、ServiceWorkerがキャッシュした古い一覧に含まれる
        // 改名前の板名を書き込まないよう、追加するものがあるときだけサーバに問い直す。
        // これが無いと「板名を直す前の名前」が毎回ツリーに足し直され、
        // 削除しても復活する板になる（タグ版と同じ対策）
        if (not_found.size !== 0) {
            const reget_req = new GetMiBoardRequest()
            reget_req.force_reget = true
            const reget_res = await GkillAPI.get_gkill_api().get_mi_board_list(reget_req)
            if (reget_res.errors && reget_res.errors.length !== 0) {
                return reget_res.errors
            }
            not_found = collect_not_found(reget_res.boards)
        }

        not_found.forEach(board_name => {
            const board_struct = new MiBoardStructElementData()
            board_struct.key = board_name
            board_struct.name = board_name
            board_struct.check_when_inited = true
            board_struct.is_checked = board_struct.check_when_inited
            board_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            board_struct.board_name = board_name
            if (!this.mi_board_struct.children) {
                this.mi_board_struct.children = []
            }
            this.mi_board_struct.children.push(board_struct)
        })
        return new Array<GkillError>()
    }

    async append_no_devices(): Promise<Array<GkillError>> {
        // 直下だけでなくツリー全体を探す。フォルダへ移動されていても重複生成しない
        const exist = has_struct_node(this.device_struct, node => (node as DeviceStructElementData).device_name === "なし")

        if (!exist) {
            const device_struct = new DeviceStructElementData()
            device_struct.key = "なし"
            device_struct.name = i18n.global.t('NO_DEVICE_NAME')
            device_struct.check_when_inited = true
            device_struct.is_checked = device_struct.check_when_inited
            device_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            device_struct.device_name = "なし"
            device_struct.is_dir = false
            if (!this.device_struct.children) {
                this.device_struct.children = []
            }
            this.device_struct.children.push(device_struct)
        }
        return new Array<GkillError>()
    }

    async append_not_found_devices(cached_rep_names?: Array<string>): Promise<Array<GkillError>> {
        const res = await resolve_rep_names(cached_rep_names)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }

        const not_found = new Set<string>()
        res.rep_names.forEach(rep_name => {
            const device_name = this.get_device_from_rep_name(rep_name)
            let exist = false
            let device_name_walk = (_device: DeviceStructElementData): void => { }
            device_name_walk = (device: DeviceStructElementData): void => {
                const rep_children = device.children
                if (device_name === (device.device_name)) {
                    exist = true
                }
                if (rep_children) {
                    rep_children.forEach(child_rep => {
                        if (child_rep) {
                            device_name_walk(child_rep)
                        }
                    })
                }
            }
            device_name_walk(this.device_struct)
            if (!exist) {
                if (device_name) {
                    not_found.add(device_name)
                }
            }
        })

        not_found.forEach(device_name => {
            const device_struct = new DeviceStructElementData()
            device_struct.key = device_name
            device_struct.name = device_name
            device_struct.check_when_inited = true
            device_struct.is_checked = device_struct.check_when_inited
            device_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            device_struct.device_name = device_name
            device_struct.is_dir = false
            if (!this.device_struct.children) {
                this.device_struct.children = []
            }
            this.device_struct.children.push(device_struct)
        })
        return new Array<GkillError>()
    }
    async append_not_found_rep_types(cached_rep_names?: Array<string>): Promise<Array<GkillError>> {
        const res = await resolve_rep_names(cached_rep_names)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }

        const rep_type_map = generate_rep_type_map()

        // 未カスタマイズ(name === rep_type_name)の既存エントリをローカライズ名に置き換える
        let localize_default_name_walk = (_rep_type: RepTypeStructElementData): void => { }
        localize_default_name_walk = (rep_type: RepTypeStructElementData): void => {
            const localized_name = rep_type_map.get(rep_type.rep_type_name)
            if (localized_name && rep_type.name === rep_type.rep_type_name) {
                rep_type.name = localized_name
            }
            rep_type.children?.forEach(child_rep => {
                if (child_rep) {
                    localize_default_name_walk(child_rep)
                }
            })
        }
        localize_default_name_walk(this.rep_type_struct)

        const not_found = new Set<string>()
        res.rep_names.forEach(rep_name => {
            const rep_type_name = this.get_rep_type_from_rep_name(rep_name)
            let exist = false
            let rep_type_name_walk = (_rep_type: RepTypeStructElementData): void => { }
            rep_type_name_walk = (rep_type: RepTypeStructElementData): void => {
                const rep_children = rep_type.children
                if (rep_type_name === rep_type.rep_type_name) {
                    exist = true
                }
                if (rep_children) {
                    rep_children.forEach(child_rep => {
                        if (child_rep) {
                            rep_type_name_walk(child_rep)
                        }
                    })
                }
            }
            rep_type_name_walk(this.rep_type_struct)
            if (!exist) {
                if (rep_type_name) {
                    not_found.add(rep_type_name)
                }
            }
        })

        not_found.forEach(rep_type => {
            const rep_type_struct = new RepTypeStructElementData()
            rep_type_struct.key = rep_type
            rep_type_struct.name = rep_type_map.get(rep_type) ?? rep_type
            rep_type_struct.check_when_inited = true
            rep_type_struct.is_checked = rep_type_struct.check_when_inited
            rep_type_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            rep_type_struct.rep_type_name = rep_type
            rep_type_struct.is_dir = false
            if (!this.rep_type_struct.children) {
                this.rep_type_struct.children = []
            }
            this.rep_type_struct.children.push(rep_type_struct)
        })
        return new Array<GkillError>()
    }

    async append_no_tags(): Promise<Array<GkillError>> {
        // 直下だけでなくツリー全体を探す。フォルダへ移動されていても重複生成しない
        const exist = has_struct_node(this.tag_struct, node => (node as TagStructElementData).tag_name === "no tags")

        if (!exist) {
            const tag_struct = new TagStructElementData()
            tag_struct.key = "no tags"
            tag_struct.name = "no tags"
            tag_struct.check_when_inited = true
            tag_struct.is_checked = tag_struct.check_when_inited
            tag_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            tag_struct.is_force_hide = false
            tag_struct.tag_name = "no tags"
            tag_struct.is_dir = false
            this.tag_struct.children?.unshift(tag_struct)
        }
        return new Array<GkillError>()
    }

    async append_all_mi_board(): Promise<Array<GkillError>> {
        // 直下だけでなくツリー全体を探す。
        // 「すべて」をフォルダへ入れると、直下だけ見ていた頃はルートに再生成されて2個になっていた
        const exist = has_struct_node(this.mi_board_struct, node => (node as MiBoardStructElementData).board_name === "すべて")

        if (!exist) {
            const board_struct = new MiBoardStructElementData()
            board_struct.key = "すべて"
            board_struct.name = i18n.global.t('ALL_MI_BOARD_NAME')
            board_struct.check_when_inited = true
            board_struct.is_checked = board_struct.check_when_inited
            board_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            board_struct.board_name = "すべて"
            this.mi_board_struct.children?.unshift(board_struct)
        }
        return new Array<GkillError>()
    }


    constructor() {
        this.is_loaded = false
        this.user_id = ""
        this.device = ""
        this.use_dark_theme = false
        this.google_map_api_key = ""
        this.rykv_image_list_column_number = 3
        this.rykv_hot_reload = false
        this.mi_default_board = ""
        this.account_is_admin = false
        this.rykv_default_period = -1
        this.mi_default_period = -1
        this.kftl_template_struct = new KFTLTemplateElementData()
        this.tag_struct = new TagStructElementData()
        this.rep_struct = new RepStructElementData()
        this.device_struct = new DeviceStructElementData()
        this.rep_type_struct = new RepTypeStructElementData()
        this.mi_board_struct = new MiBoardStructElementData()
        this.session_is_local = false
        this.urlog_bookmarklet_session = ""
        this.for_share_kyou = false
        this.is_show_share_footer = false
        this.show_tags_in_list = true
        this.default_page = "rykv"
        this.user_id = ""
        this.device = ""
        this.user_is_admin = false
        this.cache_clear_count_limit = 1001
        this.global_ip = ""
        this.private_ip = ""
        this.lan_hostname = ""
        this.global_hostname = ""
        this.version = ""
        this.build_time = new Date(0)
        this.commit_hash = ""
    }


    get_device_from_rep_name(rep_name: string): string | null {
        const splited = rep_name.split("_")
        if (splited.length !== 3) {
            return null
        }
        if (splited.length < 2) {
            return null
        }
        return splited[1]
    }

    get_rep_type_from_rep_name(rep_name: string): string | null {
        const splited = rep_name.split("_")
        if (splited.length !== 3) {
            if (!rep_name || rep_name === "") {
                return null
            }
            return rep_name
        }
        return splited[0]
    }
}

// rep名の一覧を返します。呼び出し元が既に持っていればそれをそのまま使います。
//
// append_not_found_reps / _devices / _rep_types はどれも同じ一覧が要るので、
// 個別に取ると get_all_rep_names が3回飛ぶ。
// append_not_found_infos がまとめて1回だけ取って各メソッドへ渡す。
//
// ApplicationConfig のメソッドにしないのは、この型と構造的に同じオブジェクト
// リテラルを作っている箇所が複数あり、メソッドを増やすとそれらが
// 型エラーになるため。
async function resolve_rep_names(cached_rep_names?: Array<string>): Promise<{ rep_names: Array<string>, errors: Array<GkillError> }> {
    if (cached_rep_names) {
        return { rep_names: cached_rep_names, errors: [] }
    }
    const res = await GkillAPI.get_gkill_api().get_all_rep_names(new GetAllRepNamesRequest())
    if (res.errors && res.errors.length !== 0) {
        return { rep_names: [], errors: res.errors }
    }
    return { rep_names: res.rep_names, errors: [] }
}
