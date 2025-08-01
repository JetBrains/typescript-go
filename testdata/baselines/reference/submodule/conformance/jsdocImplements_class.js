//// [tests/cases/conformance/jsdoc/jsdocImplements_class.ts] ////

//// [a.js]
class A {
    /** @return {number} */
    method() { throw new Error(); }
}
/** @implements {A} */
class B  {
    method() { return 0 }
}

/** @implements A */
class B2  {
    /** @return {string} */
    method() { return "" }
}

/** @implements {A} */
class B3  {
}


var Ns = {};
/** @implements {A} */
Ns.C1 = class {
    method() { return 11; }
}
/** @implements {A} */
var C2 = class {
    method() { return 12; }
}
var o = {
    /** @implements {A} */
    C3: class {
        method() { return 13; }
    }
}
class CC {
    /** @implements {A} */
    C4 = class {
        method() {
            return 14;
        }
    }
}

var C5;
/** @implements {A} */
Ns.C5 = C5 || class {
    method() {
        return 15;
    }
}




//// [a.d.ts]
declare class A {
    /** @return {number} */
    method(): number;
}
/** @implements {A} */
declare class B implements A {
    method(): number;
}
/** @implements A */
declare class B2 implements A {
    /** @return {string} */
    method(): string;
}
/** @implements {A} */
declare class B3 implements A {
}
declare var Ns: {
    /** @implements {A} */
    C1: {
        new (): {
            method(): number;
        };
    };
    /** @implements {A} */
    C5: any;
};
/** @implements {A} */
declare var C2: {
    new (): {
        method(): number;
    };
};
declare var o: {
    /** @implements {A} */
    C3: {
        new (): {
            method(): number;
        };
    };
};
declare class CC {
    /** @implements {A} */
    C4: {
        new (): {
            method(): number;
        };
    };
}
declare var C5: any;
