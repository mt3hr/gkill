'use strict';


export class Repository {


    user_id: string;

    device: string;

    type: string;

    file: string;

    use_to_write: boolean;

    is_execute_idf_when_reload: boolean;

    is_enable: boolean;

    constructor() {
        this.user_id = ""
        this.device = ""
        this.type = ""
        this.file = ""
        this.use_to_write = false
        this.is_execute_idf_when_reload = false
        this.is_enable = false
    }


}



