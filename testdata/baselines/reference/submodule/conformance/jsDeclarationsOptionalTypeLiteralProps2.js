//// [tests/cases/conformance/jsdoc/declarations/jsDeclarationsOptionalTypeLiteralProps2.ts] ////

//// [foo.js]
/**
 * foo
 *
 * @public
 * @param {object} opts
 * @param {number} opts.a
 * @param {number} [opts.b]
 * @param {number} [opts.c]
 * @returns {number}
 */
function foo({ a, b, c }) {
    return a + (b ?? 0) + (c ?? 0);
}


//// [foo.js]
/**
 * foo
 *
 * @public
 * @param {object} opts
 * @param {number} opts.a
 * @param {number} [opts.b]
 * @param {number} [opts.c]
 * @returns {number}
 */
function foo({ a, b, c }) {
    return a + (b ?? 0) + (c ?? 0);
}


//// [foo.d.ts]
/**
 * foo
 *
 * @public
 * @param {object} opts
 * @param {number} opts.a
 * @param {number} [opts.b]
 * @param {number} [opts.c]
 * @returns {number}
 */
declare function foo({ a, b, c }: {
    a: number;
    b?: number;
    c?: number;
}): number;
