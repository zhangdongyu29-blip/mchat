<template>
  <div class="page roles-page">
    <header class="page-header">
      <div class="title">角色列表</div>
      <div class="actions">
        <button @click="openModal">新增角色</button>
      </div>
    </header>

    <section class="panel role-listing">
      <div v-if="loading" class="empty">加载中…</div>
      <div v-else-if="roles.length === 0" class="empty">还没有角色，点击右上角新增</div>
      <div v-else class="role-grid">
        <div
          v-for="r in roles"
          :key="r.id"
          class="role-card"
        >
          <div
            class="swipe-track"
            :style="{ transform: `translateX(${swipeOffsets[r.id] || 0}px)` }"
            @pointerdown="onSwipeStart(r.id, $event)"
            @pointermove="onSwipeMove(r.id, $event)"
            @pointerup="onSwipeEnd(r.id, $event)"
            @pointercancel="onSwipeEnd(r.id, $event)"
            @pointerleave="onSwipeEnd(r.id, $event)"
            @click="onCardClick(r.id)"
          >
            <div class="role-name">{{ r.name }}</div>
            <div class="role-meta">{{ r.style || '默认语气' }}</div>
            <div class="role-meta small ellipsis">{{ r.background || '无背景描述' }}</div>
            <div class="card-actions">
              <div class="pill gray">{{ r.call_me ? '称呼我：' + r.call_me : '未设置称呼' }}</div>
            </div>
          </div>
          <button
            class="swipe-delete"
            :class="{ visible: (swipeOffsets[r.id] || 0) <= -60 }"
            @click.stop="removeRole(r.id)"
          >删除</button>
        </div>
      </div>
    </section>

    <div v-if="showModal" class="modal-mask" @click.self="closeModal">
      <div class="modal">
        <h3>新增角色</h3>
        <form class="form" @submit.prevent="create">
          <label>角色名称</label>
          <input v-model="form.name" required placeholder="如：元气助手" />

          <label>背景设定</label>
          <textarea v-model="form.background" rows="2" placeholder="身份、世界观等"></textarea>

          <label>聊天方式 / 语气</label>
          <textarea v-model="form.style" rows="2" placeholder="温柔、简洁、技术流…"></textarea>

          <label>附加设定</label>
          <textarea v-model="form.persona_hint" rows="2" placeholder="限制、特殊提示等"></textarea>

          <label>让TA称呼我</label>
          <input v-model="form.call_me" placeholder="如：小明/老板/同学" />

          <div class="modal-actions">
            <button type="button" class="ghost" @click="closeModal">取消</button>
            <button type="submit" :disabled="submitting">保存</button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showDeleteConfirm" class="modal-mask" @click.self="cancelRemove">
      <div class="modal">
        <h3>确认删除</h3>
        <p>删除后将清空该角色的对话与记忆，确认删除？</p>
        <div class="modal-actions">
          <button class="ghost" @click="cancelRemove">取消</button>
          <button class="danger" @click="confirmRemove">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'

const router = useRouter()
const roles = ref([])
const loading = ref(false)
const submitting = ref(false)
const showModal = ref(false)
const showDeleteConfirm = ref(false)
const deleteTargetId = ref(null)
const swipeOffsets = ref({})
const swipeStartX = ref(null)
const swipeActiveId = ref(null)
const swipePointerId = ref(null)
const form = ref({
  name: '',
  background: '',
  style: '',
  persona_hint: '',
  call_me: '',
})

const load = async () => {
  loading.value = true
  try {
    roles.value = await api.listRoles()
  } finally {
    loading.value = false
  }
}

const openModal = () => {
  showModal.value = true
}
const closeModal = () => {
  showModal.value = false
}

const create = async () => {
  submitting.value = true
  try {
    await api.createRole({
      name: form.value.name,
      background: form.value.background,
      style: form.value.style,
      persona_hint: form.value.persona_hint,
      call_me: form.value.call_me,
    })
    closeModal()
    form.value = { name: '', background: '', style: '', persona_hint: '', call_me: '' }
    await load()
  } catch (e) {
    alert(e.message)
  } finally {
    submitting.value = false
  }
}

const removeRole = (id) => {
  showDeleteConfirm.value = true
  deleteTargetId.value = id
}

const cancelRemove = () => {
  showDeleteConfirm.value = false
  deleteTargetId.value = null
}

const confirmRemove = async () => {
  if (!deleteTargetId.value) return
  await api.deleteRole(deleteTargetId.value)
  await load()
  cancelRemove()
}

const goChat = (id) => {
  router.push({ path: '/chat', query: { roleId: id } })
}

const onSwipeStart = (id, e) => {
  // 收起其他卡片
  swipeOffsets.value = { [id]: 0 }
  swipeActiveId.value = id
  swipeStartX.value = e.clientX
  swipePointerId.value = e.pointerId
  if (e.target?.setPointerCapture) {
    e.target.setPointerCapture(e.pointerId)
  }
}
const onSwipeMove = (id, e) => {
  if (swipeActiveId.value !== id || swipeStartX.value === null) return
  const delta = e.clientX - swipeStartX.value
  // only allow left swipe
  const offset = Math.max(-100, Math.min(0, delta))
  swipeOffsets.value = { ...swipeOffsets.value, [id]: offset }
}
const onSwipeEnd = (id, e) => {
  if (swipeActiveId.value !== id) return
  const current = swipeOffsets.value[id] || 0
  const finalOffset = current <= -60 ? -90 : 0
  swipeOffsets.value = { ...swipeOffsets.value, [id]: finalOffset }
  if (swipePointerId.value !== null && e.target?.releasePointerCapture) {
    try { e.target.releasePointerCapture(swipePointerId.value) } catch {}
  }
  swipeActiveId.value = null
  swipeStartX.value = null
  swipePointerId.value = null
}
const onCardClick = (id) => {
  const offset = swipeOffsets.value[id] || 0
  if (offset < -10) {
    // tap to close swipe
    swipeOffsets.value = { ...swipeOffsets.value, [id]: 0 }
    return
  }
  goChat(id)
}

onMounted(load)
</script>
