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
