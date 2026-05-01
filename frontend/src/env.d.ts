/// <reference types="vite/client" />

declare const __BUILD_NUMBER__: string
declare const __BUILD_TIME__: string
declare const __COMMIT_SHA__: string

interface ImportMetaEnv {
  readonly VITE_GA_ID?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
