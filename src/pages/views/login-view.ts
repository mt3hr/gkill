// ˅
'use strict';

import { LoginViewEmits } from './login-view-emits';
import { LoginViewProps } from './login-view-props';

// ˄

export class LoginView {
    // ˅
    
    // ˄

    private user_id: Ref<string>;

    private password: Ref<string>;

    private password_sha256: Ref<string>;

    private loginViewProps: LoginViewProps;

    private loginViewEmits: LoginViewEmits;

    try_login(user_id: string, password_sha256: string): boolean {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
