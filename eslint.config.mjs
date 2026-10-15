import js from "@eslint/js";
import tseslint from "typescript-eslint";
import { defineConfig } from "eslint/config";

export default defineConfig({
  files: ["app/web/src/**/*.ts", "app/web/test/**/*.ts"],
  extends: [js.configs.recommended, tseslint.configs.recommended],
});
