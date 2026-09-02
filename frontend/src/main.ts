import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ArcoVue from '@arco-design/web-vue'
import '@arco-design/web-vue/dist/arco.css'
import router, { startRoutePermWatcher } from './router'
import App from './App.vue'
import './tailwind.css'
import './style.scss'
import vPerm from './directives/permission'

const app = createApp(App)
app.directive('perm', vPerm)
app.use(ArcoVue)
app.use(createPinia())
app.use(router)
startRoutePermWatcher()
app.mount('#app')
