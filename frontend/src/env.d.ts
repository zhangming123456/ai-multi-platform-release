/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

declare module 'swagger-ui-dist/swagger-ui-bundle.js' {
  const SwaggerUIBundle: any
  export default SwaggerUIBundle
}

declare module 'swagger-ui-dist/swagger-ui.css' {}
