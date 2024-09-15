'use strict';


export class LoginRequest {


    user_id: string;

    password_sha256: string;

    constructor() {
        this.user_id = "";
        this.password_sha256 = "";

    }


}



