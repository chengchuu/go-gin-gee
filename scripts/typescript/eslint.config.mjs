import babelParser from "@babel/eslint-parser";
import stylistic from "@stylistic/eslint-plugin";

export default [
  { ignores: [ "dist/**" ] },
  {
    files: [ "src/**/*.ts", "test/**/*.mjs", "eslint.config.mjs" ],
    plugins: { "@stylistic": stylistic },
    rules: {
      "@stylistic/semi": [ "warn", "always" ],
      "@stylistic/quotes": [ "warn", "double" ],
      "@stylistic/indent": [ "warn", 2, { SwitchCase: 1 } ],
      "@stylistic/comma-dangle": [ "warn", "always-multiline" ],
      "@stylistic/eol-last": [ "warn", "always" ],
      "@stylistic/spaced-comment": [ "warn", "always" ],
      "@stylistic/object-curly-spacing": [ "warn", "always" ],
      "@stylistic/array-bracket-spacing": [ "warn", "always" ],
      "@stylistic/object-curly-newline": [
        "warn",
        { ImportDeclaration: { multiline: true, minProperties: 4 } },
      ],
    },
  },
  {
    files: [ "src/**/*.ts" ],
    languageOptions: {
      parser: babelParser,
      parserOptions: {
        requireConfigFile: false,
        babelOptions: {
          babelrc: false,
          configFile: false,
          parserOpts: { plugins: [ "typescript" ] },
        },
      },
    },
  },
];
