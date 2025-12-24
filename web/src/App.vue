<template>
  <div class="app-shell">
    <router-view />
    <nav class="tabbar">
      <button :class="{ active: current === 'roles' }" @click="go('/roles')">角色</button>
      <button :class="{ active: current === 'chat' }" @click="go('/chat')">聊天</button>
    </nav>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const router = useRouter()
const route = useRoute()

const current = computed(() => {
  if (route.path.startsWith('/chat')) return 'chat'
  return 'roles'
})

const go = (path) => {
  if (route.path !== path) router.push(path)
}
</script>
