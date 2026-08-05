const is_array = Array.isArray;
const key_list = Object.keys;
const has_prop = Object.prototype.hasOwnProperty;

export function deep_equals<T>(a: T, b: T): boolean {
    if (a === b) { return true; }


    if (a && b && typeof a === 'object' && typeof b === 'object') {
        const arr_a = is_array(a)
            , arr_b = is_array(b);
        let i
            , length
            , key;

        if (arr_a && arr_b) {
            length = a.length;
            if (length !== b.length) { return false; }
            for (i = length; i-- !== 0;) {
                if (!deep_equals(a[i], b[i])) {
                    return false;
                }
            }
            return true;
        }

        if (arr_a !== arr_b) { return false; }

        const date_a = a instanceof Date
            , date_b = b instanceof Date;
        if (date_a !== date_b) { return false; }
        if (date_a && date_b) { return a.getTime() === b.getTime(); }

        const regexp_a = a instanceof RegExp
            , regexp_b = b instanceof RegExp;
        if (regexp_a !== regexp_b) { return false; }
        if (regexp_a && regexp_b) { return a.toString() === b.toString(); }

        const keys = key_list(a);
        length = keys.length;

        if (length !== key_list(b).length) {
            return false;
        }

        for (i = length; i-- !== 0;) {
            if (!has_prop.call(b, keys[i])) {
                return false;
            }
        }

        for (i = length; i-- !== 0;) {
            key = keys[i];
            if (!deep_equals((a as Record<string, unknown>)[key], (b as Record<string, unknown>)[key])) { return false; }
        }

        return true;
    }

    return a !== a && b !== b;
}