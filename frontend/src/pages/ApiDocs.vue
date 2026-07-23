<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageHeader from '@/components/layout/PageHeader.vue'

const containerRef = ref<HTMLDivElement | null>(null)

onMounted(async () => {
  const SwaggerUI = (await import('swagger-ui-dist/swagger-ui-bundle.js')).default
  await import('swagger-ui-dist/swagger-ui.css')
  SwaggerUI({
    dom_id: '#swagger-ui-container',
    url: '/openapi.json',
    presets: [SwaggerUI.presets.apis, SwaggerUI.SwaggerUIStandalonePreset],
    layout: 'BaseLayout',
    deepLinking: true,
    docExpansion: 'list',
  })
})
</script>

<template>
  <div class="flex flex-col h-full">
    <PageHeader title="API 文档" subtitle="在线接口调试与文档" />
    <div class="flex-1 overflow-auto p-6">
      <div
        id="swagger-ui-container"
        ref="containerRef"
        class="bg-white rounded-[16px] border border-black/[0.06] shadow-sm"
      />
    </div>
  </div>
</template>

<style>
#swagger-ui-container .swagger-ui {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial,
    sans-serif;
}
#swagger-ui-container .swagger-ui .topbar {
  display: none;
}
#swagger-ui-container .swagger-ui .info {
  margin: 20px 0;
}
#swagger-ui-container .swagger-ui .scheme-container {
  margin: 0 0 20px;
  padding: 20px;
  background: #fafafa;
  border-radius: 12px;
}

@media (max-width: 248px) {
  #swagger-ui-container .swagger-ui {
    font-size: 11px !important;
  }
  #swagger-ui-container .swagger-ui .info {
    margin: 10px 0 !important;
  }
  #swagger-ui-container .swagger-ui .info h2 {
    font-size: 14px !important;
  }
  #swagger-ui-container .swagger-ui .opblock-tag {
    font-size: 12px !important;
    padding: 6px 8px !important;
  }
  #swagger-ui-container .swagger-ui .opblock-summary {
    padding: 6px 8px !important;
  }
  #swagger-ui-container .swagger-ui .opblock-summary-method {
    font-size: 10px !important;
    padding: 2px 4px !important;
    min-width: 36px !important;
  }
  #swagger-ui-container .swagger-ui .opblock-summary-path {
    font-size: 11px !important;
  }
}
</style>
