<template>
  <div class="page">
    <header class="topbar">
      <div class="title">mchat · H5 小程序风格</div>
      <div class="actions">
        <button class="ghost" @click="fetchRoles">刷新角色</button>
        <button class="ghost" @click="fetchMemories" :disabled="!activeRoleId">加载记忆</button>
      </div>
    </header>

    <main class="layout">
      <section class="panel roles">
        <h3>创建角色</h3>
        <form class="form" @submit.prevent="createRole">
          <label>角色名称</label>
          <input v-model="roleForm.name" placeholder="例如：元气助手" required />

          <label>背景设定</label>
          <textarea v-model="roleForm.background" rows="2" placeholder="身份、世界观等"></textarea>

          <label>聊天方式 / 语气</label>
          <textarea v-model="roleForm.style" rows="2" placeholder="温柔、简洁、技术流…"></textarea>

          <label>附加设定</label>
          <textarea v-model="roleForm.persona_hint" rows="2" placeholder="限制、特殊提示等"></textarea>

          <button type="submit">创建</button>
        </form>

        <h3>角色列表</h3>
        <div class="role-list">
          <div
            v-for="r in roles"
            :key="r.id"
            class="role-card"
            :class="{ active: r.id === activeRoleId }"
            @click="selectRole(r.id)"
          >
            <div class="name">{{ r.name }}</div>
            <div class="meta">{{ r.style || '默认语气' }}</div>
          </div>
        </div>
      </section>

      <section class="panel chat">
        <div class="chat-header">
          <div class="heading">聊天窗口</div>
          <div class="role-status" v-if="activeRole">
            当前：<strong>{{ activeRole.name }}</strong>
          </div>
          <div class="role-status" v-else>请选择一个角色开始聊天</div>
        </div>

        <div class="chat-box">
          <div v-if="messages.length === 0" class="empty">暂无聊天记录</div>
          <div v-else class="chat-messages">
            <div v-for="m in messages" :key="m.id" class="bubble" :data-from="m.sender">
              <div class="sender">{{ senderLabel(m.sender) }}</div>
              <div class="content">{{ m.content }}</div>
            </div>
          </div>
        </div>

        <form class="input-bar" @submit.prevent="sendMessage">
          <input
            v-model="chatInput"
            :disabled="!activeRoleId || sending"
            placeholder="输入消息…"
            required
          />
          <button type="submit" :disabled="!activeRoleId || sending">发送</button>
        </form>

        <div class="memories">
          <div class="heading">记忆摘要</div>
          <div v-if="memories.length === 0" class="empty">暂无记忆</div>
          <ul v-else>
            <li v-for="m in memories" :key="m.id">
              <div class="time">{{ new Date(m.created_at).toLocaleString() }}</div>
              <div class="text">{{ m.summary }}</div>
            </li>
          </ul>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'

const base = import.meta.env.VITE_API_BASE || ''
const api = axios.create({
  baseURL: base ? `${base.replace(/\/$/, '')}/api` : '/api',
})

const roles = ref([])
const messages = ref([])
const memories = ref([])
const activeRoleId = ref(null)
const sending = ref(false)
const chatInput = ref('')
const roleForm = ref({
  name: '',
  background: '',
  style: '',
  persona_hint: '',
})

const activeRole = computed(() => roles.value.find(r => r.id === activeRoleId.value))

const senderLabel = (sender) => {
  if (sender === 'ai') return 'AI'
  if (sender === 'system') return '系统'
  return '我'
}

const fetchRoles = async () => {
  const { data } = await api.get('/roles')
  roles.value = data
  if (!activeRoleId.value && data.length > 0) {
    activeRoleId.value = data[0].id
    await fetchMessages()
    await fetchMemories()
  }
}

const fetchMessages = async () => {
  if (!activeRoleId.value) return
  const { data } = await api.get(`/chat/${activeRoleId.value}`)
  messages.value = data
}

const fetchMemories = async () => {
  if (!activeRoleId.value) return
  const { data } = await api.get(`/memories/${activeRoleId.value}`)
  memories.value = data
}

const selectRole = async (id) => {
  activeRoleId.value = id
  await fetchMessages()
  await fetchMemories()
}

const createRole = async () => {
  await api.post('/roles', roleForm.value)
  roleForm.value = { name: '', background: '', style: '', persona_hint: '' }
  await fetchRoles()
}

const sendMessage = async () => {
  if (!chatInput.value.trim() || !activeRoleId.value) return
  sending.value = true
  try {
    const { data } = await api.post('/chat', {
      role_id: activeRoleId.value,
      message: chatInput.value,
    })
    messages.value.push({
      id: Date.now(),
      sender: 'user',
      content: chatInput.value,
    })
    messages.value.push({
      id: Date.now() + 1,
      sender: 'ai',
      content: data.reply,
    })
    chatInput.value = ''
    await fetchMemories()
  } catch (e) {
    alert(e.response?.data?.error || e.message)
  } finally {
    sending.value = false
  }
}

onMounted(fetchRoles)
</script>
