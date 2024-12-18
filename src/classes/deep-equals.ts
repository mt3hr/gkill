export function deepEquals(d1:any, d2:any){
    function toKV(o:any){
        return Object.entries(o).sort(([k1], [k2])=>{
            let i = 0;

            while(k1[i] === k2[i]){
                if(k1[i] === undefined && k2[i] === undefined){
                    throw "Same property name.";
                }

                i++;
            }

            return (k1.codePointAt(i) ?? 0) - (k2.codePointAt(i) ?? 0);
        });
    }

    if(d1 === d2){
        return true;
    }

    if(typeof d1 !== "object" || typeof d2 !== "object" || d1 === null || d2 === null){
        return false;
    }

    const kv1 = toKV(d1);
    const kv2 = toKV(d2);

    if(kv1.length !== kv2.length){
        return false;
    }

    for(let i = 0; i < kv1.length; i++){
        const [k1, v1] = kv1[i];
        const [k2, v2] = kv2[i];

        if(k1 !== k2 || !deepEquals(v1, v2)){
            return false;
        }
    }

    return true;
}