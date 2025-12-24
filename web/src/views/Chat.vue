<template>
  <div class="page chat-page">
    <div v-if="errorMsg" class="toast error">{{ errorMsg }}</div>
    <header class="page-header">
      <div class="title">
        <span v-if="currentRole">{{ currentRole.name }}</span>
        <span v-else>请选择角色</span>
      </div>
      <div class="actions">
        <button class="ghost" @click="openRolePanel" :disabled="!currentRole">角色设定</button>
        <button class="ghost" @click="refreshAll" :disabled="loading">刷新</button>
      </div>
    </header>

    <div class="chat-layout">
      <aside class="sidebar" :class="{ collapsed: sidebarCollapsed }">
        <div class="sidebar-header">
          <div class="label">会话</div>
          <div class="side-actions">
            <button class="ghost small" @click="openConvModal" :disabled="!currentRole">会话列表</button>
            <button class="small" @click="createConversation" :disabled="!currentRole">新对话</button>
          </div>
        </div>
      </aside>

      <section class="chat-panel">
        <div class="chat-toolbar"></div>

        <div class="chat-box" ref="chatBoxRef">
          <div v-if="messagesLoading" class="empty">加载消息中…</div>
          <div v-else-if="messages.length === 0" class="empty">暂无聊天记录</div>
          <div v-else class="chat-messages">
            <div
              v-for="m in messages"
              :key="m.id"
              class="bubble"
              :data-from="m.sender"
              :class="{ streaming: m.streaming }"
            >
              <div class="sender">{{ senderLabel(m.sender) }}</div>
              <div class="content">
                <template v-if="m.streaming && !m.content">
                  <span class="dot-flash"></span>
                  <span class="dot-flash"></span>
                  <span class="dot-flash"></span>
                </template>
                <template v-else>
                  {{ m.content }}
                  <span v-if="m.streaming" class="caret">▋</span>
                </template>
              </div>
            </div>
          </div>
        </div>

        <form class="input-bar" @submit.prevent="sendMessage">
          <input
            v-model="chatInput"
            :disabled="!currentRole || sending"
            placeholder="输入消息…"
            required
          />
          <button type="submit" :disabled="!currentRole || sending">{{ sending ? '发送中…' : '发送' }}</button>
        </form>
      </section>
    </div>

    <div v-if="showRolePanel" class="modal-mask" @click.self="closeRolePanel">
      <div class="modal wide">
        <h3>角色设定</h3>
        <form class="form" @submit.prevent="updateRole">
          <label>角色名称</label>
          <input v-model="roleForm.name" required />

          <label>背景设定</label>
          <textarea v-model="roleForm.background" rows="2"></textarea>

          <label>聊天方式 / 语气</label>
          <textarea v-model="roleForm.style" rows="2"></textarea>

          <label>附加设定</label>
          <textarea v-model="roleForm.persona_hint" rows="2"></textarea>

          <label>让TA称呼我</label>
          <input v-model="roleForm.call_me" />

          <div class="modal-actions">
            <button type="button" class="ghost" @click="closeRolePanel">关闭</button>
            <button type="submit" :disabled="roleSaving">保存</button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showConvModal" class="modal-mask" @click.self="closeConvModal">
      <div class="modal wide">
        <h3>选择会话</h3>
        <div class="conversation-list modal-list">
          <div v-if="convLoading" class="empty">加载中…</div>
          <div v-else-if="conversations.length === 0" class="empty">暂无对话</div>
          <div
            v-else
            v-for="c in conversations"
            :key="c.id"
            class="conversation-item"
            :class="{ active: c.id === conversationId }"
            @click="handleSelectInModal(c.id)"
          >
            <div class="c-title">{{ c.title || '新的对话' }}</div>
            <button class="icon-btn danger-text" @click.stop="removeConversation(c.id)">❌</button>
          </div>
        </div>
        <div class="modal-actions">
          <button class="ghost" @click="closeConvModal">关闭</button>
        </div>
      </div>
    </div>

    <div v-if="showConvDeleteConfirm" class="modal-mask" @click.self="cancelDeleteConversation">
      <div class="modal">
        <h3>确认删除</h3>
        <p>删除后将清空该对话内的消息，确认删除？</p>
        <div class="modal-actions">
          <button class="ghost" @click="cancelDeleteConversation">取消</button>
          <button class="danger" @click="confirmDeleteConversation">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, apiBase, chatStream } from '../api'

const router = useRouter()
const route = useRoute()

const roles = ref([])
const currentRoleId = ref(null)
const conversations = ref([])
const conversationId = ref(null)
const newConversationPending = ref(false)
const messages = ref([])
const messagesLoading = ref(false)
const chatInput = ref('')
const sending = ref(false)
const streaming = ref(false)
const sidebarCollapsed = ref(false)
const loading = ref(false)
const convLoading = ref(false)
const errorMsg = ref('')
const showRolePanel = ref(false)
const showConvModal = ref(false)
const showConvDeleteConfirm = ref(false)
const deleteConvId = ref(null)
const roleSaving = ref(false)
const roleForm = ref({
  name: '',
  background: '',
  style: '',
  persona_hint: '',
  call_me: '',
})

const currentRole = computed(() => roles.value.find(r => r.id === Number(currentRoleId.value)))
const currentConversation = computed(() => conversations.value.find(c => c.id === Number(conversationId.value)))

const senderLabel = (sender) => {
  if (sender === 'ai') return currentRole.value?.name || 'AI'
  if (sender === 'system') return '系统'
  return roleForm.value.call_me || '我'
}

const loadRoles = async () => {
  loading.value = true
  try {
    roles.value = await api.listRoles()
    if (!currentRoleId.value && roles.value.length > 0) {
      const qRole = Number(route.query.roleId)
      currentRoleId.value = qRole || roles.value[0].id
    }
  } finally {
    loading.value = false
  }
}

const loadRoleDetail = async () => {
  if (!currentRoleId.value) return
  const detail = await api.getRole(currentRoleId.value)
  roleForm.value = {
    name: detail.name || '',
    background: detail.background || '',
    style: detail.style || '',
    persona_hint: detail.persona_hint || '',
    call_me: detail.call_me || '',
  }
}

const loadConversations = async () => {
  if (!currentRoleId.value) return
  convLoading.value = true
  try {
    conversations.value = await api.listConversations(currentRoleId.value)
    if (!conversationId.value && conversations.value.length > 0) {
      conversationId.value = conversations.value[0].id
    }
    if (conversations.value.length === 0 && !newConversationPending.value) {
      conversationId.value = null
      messages.value = []
    }
  } finally {
    convLoading.value = false
  }
}

const loadMessages = async () => {
  messagesLoading.value = true
  try {
    if (!conversationId.value) {
      messages.value = []
      return
    }
    messages.value = await api.listMessages(conversationId.value)
    scrollToBottom()
  } finally {
    messagesLoading.value = false
  }
}

const refreshAll = async () => {
  await loadRoles()
  await loadRoleDetail()
  await loadConversations()
  await loadMessages()
}

onMounted(async () => {
  if (window.innerWidth < 900) {
    sidebarCollapsed.value = true
  }
  await loadRoles()
  await loadRoleDetail()
  await loadConversations()
  await loadMessages()
})

watch(currentRoleId, async () => {
  await loadRoleDetail()
  await loadConversations()
  await loadMessages()
  router.replace({ path: '/chat', query: { roleId: currentRoleId.value } })
})

const selectConversation = async (id) => {
  conversationId.value = id
  newConversationPending.value = false
  await loadMessages()
}

const createConversation = async () => {
  if (!currentRoleId.value) return
  newConversationPending.value = true
  conversationId.value = null
  messages.value = []
}

const removeConversation = async (id) => {
  showConvDeleteConfirm.value = true
  deleteConvId.value = id
}

const sendMessage = async () => {
  if (!chatInput.value.trim() || !currentRoleId.value) return
  const text = chatInput.value
  chatInput.value = ''
  sending.value = true
  streaming.value = true

  // ensure conversation
  // create on first send only when pending
  if (!conversationId.value && !newConversationPending.value) {
    newConversationPending.value = true
  }

  messages.value.push({
    id: Date.now(),
    sender: 'user',
    content: text,
  })
  const streamingId = Date.now() + 1
  messages.value.push({
    id: streamingId,
    sender: 'ai',
    content: '',
    streaming: true,
  })

  try {
    await streamChatFlow(text, streamingId)
  } catch (e) {
    if (newConversationPending.value && conversationId.value) {
      try {
        await api.deleteConversation(conversationId.value)
      } catch (err) {
        console.warn('cleanup failed', err)
      }
      conversationId.value = null
      newConversationPending.value = false
    }
    setError(e.message)
  } finally {
    sending.value = false
    streaming.value = false
  }
}

const streamChatFlow = async (text, streamingId) => {
  const res = await chatStream({
    roleId: currentRoleId.value,
    conversationId: conversationId.value,
    message: text,
  })
  if (!res.ok || !res.body) {
    throw new Error(`请求失败 ${res.status}`)
  }

  const newConvoId = res.headers.get('x-conversation-id')
  if (newConvoId) {
    conversationId.value = Number(newConvoId)
    newConversationPending.value = false
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let done = false
  while (!done) {
    const { value, done: streamDone } = await reader.read()
    done = streamDone
    if (value) {
      const chunk = decoder.decode(value, { stream: true })
      appendStreaming(streamingId, chunk)
    }
  }
  finalizeStreaming(streamingId)
  await loadConversations()
  await loadMessages()
}

const appendStreaming = (streamingId, chunk) => {
  const idx = messages.value.findIndex(m => m.id === streamingId)
  if (idx === -1) return
  messages.value[idx].content += chunk
  scrollToBottom()
}

const chatBoxRef = ref(null)

const finalizeStreaming = (streamingId) => {
  const idx = messages.value.findIndex(m => m.id === streamingId)
  if (idx === -1) return
  messages.value[idx].streaming = false
  scrollToBottom()
}

const scrollToBottom = async () => {
  await nextTick()
  const box = chatBoxRef.value
  if (box) {
    box.scrollTop = box.scrollHeight
  }
}

watch(
  () => messages.value.length,
  () => {
    scrollToBottom()
  }
)

const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

const openConvModal = async () => {
  showConvModal.value = true
  await loadConversations()
}
const closeConvModal = () => {
  showConvModal.value = false
}
const handleSelectInModal = async (id) => {
  await selectConversation(id)
  closeConvModal()
}

const confirmDeleteConversation = async () => {
  if (!deleteConvId.value) return
  await api.deleteConversation(deleteConvId.value)
  await loadConversations()
  await loadMessages()
  showConvDeleteConfirm.value = false
  deleteConvId.value = null
}

const cancelDeleteConversation = () => {
  showConvDeleteConfirm.value = false
  deleteConvId.value = null
}

const openRolePanel = () => {
  if (!currentRoleId.value) return
  showRolePanel.value = true
}
const closeRolePanel = () => {
  showRolePanel.value = false
}

const updateRole = async () => {
  roleSaving.value = true
  try {
    await api.updateRole(currentRoleId.value, roleForm.value)
    await loadRoles()
  } catch (e) {
    setError(e.message)
  } finally {
    roleSaving.value = false
    closeRolePanel()
  }
}

const setError = (msg) => {
  errorMsg.value = msg || '请求出错'
  setTimeout(() => {
    errorMsg.value = ''
  }, 2500)
}
</script>
