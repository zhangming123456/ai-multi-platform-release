import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ArcoVue from '@arco-design/web-vue'
import '@arco-design/web-vue/dist/arco.css'
import router from './router'
import App from './App.vue'
import './style.css'
import vPerm from './directives/permission'

const app = createApp(App)
app.directive('perm', vPerm)
app.use(ArcoVue)
app.use(createPinia())
app.use(router)
app.mount('#app')
