import { createRouter, createWebHistory } from 'vue-router'
import Roles from '../views/Roles.vue'
import Chat from '../views/Chat.vue'

const routes = [
  { path: '/', redirect: '/roles' },
  { path: '/roles', component: Roles },
  { path: '/chat', component: Chat },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
