/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type, @typescript-eslint/no-explicit-any
  const component: DefineComponent<{}, {}, any>
  export default component
}

declare module 'swagger-ui-dist/swagger-ui-bundle.js' {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const SwaggerUI: any
  export default SwaggerUI
}

declare module 'vite/client' {
  interface ImportMetaEnv {
    readonly VITE_EDITOR_ENGINE?: 'lightweight' | 'powerful'
  }

  interface ImportMeta {
    readonly env: ImportMetaEnv
  }
}
