/**
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

// function __parseExports(exports) {
//   return Object.fromEntries(
//     Object
//       .entries(exports)
//       .map(([name, { fn, args, ...rest }]) => {
//         if (!fn) {
//           throw new Error(`Function ${name} is missing the 'fn' function declaration.`);
//         }

//         if (!args) {
//           throw new Error(`Function ${name} is missing the 'args' arguments declaration.`);
//         }

//         const fnString = fn.toString();
//         const argNames = fnString
//           .match(/\(([^)]*)\)/)[1]
//           .split(",")
//           .map(arg => arg.trim())
//           .filter(arg => arg);

//         return [
//           name,
//           {
//             fn: fn,
//             fnString: fn?.toString(),
//             args: args.map((arg, index) => ({...arg, name: argNames[index]})),
//             ...rest
//           }
//         ];
//       })
//   );
// }


function $__parseExports(exports) {
  return Object.fromEntries(
    Object
      .entries(exports)
      .map(([name, { fn, ...rest }]) => {
        return [
          name,
          {
            fn: fn,
            fnString: fn?.toString(),
            ...rest
          }
        ];
      })
  );
}

const __exports = $__parseExports(exports);
