'use strict';


export class RepTypeStructElementData {


    id: string;

    rep_type_name: string;

    check_when_inited: boolean;

    children: Array<RepTypeStructElementData> | null

    constructor() {
        this.id = ""
        this.rep_type_name = ""
        this.check_when_inited = false
        this.children = null
    }


}



