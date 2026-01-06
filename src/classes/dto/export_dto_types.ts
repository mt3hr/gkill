export type ExportKyouDTO = {
    meta: {
        id: string
        data_type: string
        related_time: string
        create_time?: string
        update_time?: string
        is_deleted?: boolean
        image_source?: string
    }

    tags?: string[]
    texts?: TextDTO[]
    notifications?: NotificationDTO[]

    payload:
    | TimeIsDTO
    | KmemoDTO
    | KcDTO
    | UrlogDTO
    | NlogDTO
    | MiDTO
    | LantanaDTO
    | IdfKyouDTO
    | GitCommitLogDTO
    | ReKyouDTO
}

export type TextDTO = {
    id: string
    text: string
}

export type NotificationDTO = {
    id: string
    content: string
    notification_time?: string
    is_notificated?: boolean
}

export type TimeIsDTO = {
    kind: "timeis"
    title: string
    start_time: string
    end_time?: string
}

export type KmemoDTO = {
    kind: "kmemo"
    content: string
}

export type KcDTO = {
    kind: "kc"
    key: string
    value: number
}

export type UrlogDTO = {
    kind: "urlog"
    title: string
    url: string
}

export type NlogDTO = {
    kind: "nlog"
    title: string
    shop?: string
    amount: number
}

export type MiDTO = {
    kind: "mi"
    title: string
    is_checked: boolean
    board_name?: string
    limit_time?: string
    estimate_start_time?: string
    estimate_end_time?: string
}

export type LantanaDTO = {
    kind: "lantana"
    mood: number
}

export type IdfKyouDTO = {
    kind: "idf"
    file_path: string
}

export type GitCommitLogDTO = {
    kind: "git"
    commit_message: string
    addition?: number
    deletion?: number
}

export type ReKyouDTO = {
    kind: "rekyou"
    from_id: string
    to_id: string
}
