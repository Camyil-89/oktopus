import packageJson from "./package.json";

/** Product version (SemVer); source of truth: web/package.json */
export const APP_VERSION = packageJson.version;
