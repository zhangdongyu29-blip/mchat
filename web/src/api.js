const base = import.meta.env.VITE_API_BASE || ''
export const apiBase = base ? base.replace(/\/$/, '') : ''

const getBase = (path) => (apiBase ? `${apiBase}${path}` : path)

export async function request(path, options = {}) {
  const res = await fetch(getBase(path), {
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    ...options,
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(text || `请求失败 ${res.status}`)
  }
  if (res.status === 204) return null
  const ct = res.headers.get('Content-Type') || ''
  if (ct.includes('application/json')) return res.json()
  return res.text()
}

export const api = {
  listRoles: () => request('/api/roles'),
  createRole: (payload) => request('/api/roles', { method: 'POST', body: JSON.stringify(payload) }),
  getRole: (id) => request(`/api/roles/${id}`),
  updateRole: (id, payload) => request(`/api/roles/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteRole: (id) => request(`/api/roles/${id}`, { method: 'DELETE' }),
  listConversations: (roleId) => request(`/api/roles/${roleId}/conversations`),
  createConversation: (roleId, payload) =>
    request(`/api/roles/${roleId}/conversations`, { method: 'POST', body: JSON.stringify(payload) }),
  deleteConversation: (id) => request(`/api/conversations/${id}`, { method: 'DELETE' }),
  listMessages: (conversationId) => request(`/api/conversations/${conversationId}/messages`),
  listMemories: (roleId) => request(`/api/memories/${roleId}`),
}

export function chatStream({ roleId, conversationId, message }) {
  const url = getBase('/api/chat')
  return fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ role_id: roleId, conversation_id: conversationId || 0, message }),
  })
}
