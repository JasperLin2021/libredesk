import { defineStore } from 'pinia'
import { ref } from 'vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '../api'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'

// Pin source discriminator used in both the API path and the payload. It
// must stay in sync with the `source` column constraint on
// `user_canned_pins` and the PinSource enum on the Go side.
export const PIN_SOURCE = {
  GLOBAL: 'global',
  PERSONAL: 'personal'
}

export const useCannedResponsesStore = defineStore('cannedResponses', () => {
  // The aggregate payload returned by the agent endpoint. The panel reads
  // from these refs and never makes its own ad-hoc fetches.
  const pinned = ref([])
  const personal = ref([])
  const globalTree = ref([])
  const loading = ref(false)
  const loaded = ref(false)
  const emitter = useEmitter()

  const toastError = (error) => {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }

  const fetch = async (force = false) => {
    if (loading.value) return
    if (loaded.value && !force) return
    loading.value = true
    try {
      const resp = await api.getAgentCannedResponses()
      const data = resp?.data?.data || { pinned: [], personal: [], global: [] }
      pinned.value = data.pinned || []
      personal.value = data.personal || []
      globalTree.value = data.global || []
      loaded.value = true
    } catch (error) {
      toastError(error)
    } finally {
      loading.value = false
    }
  }

  // Personal CRUD keeps the store consistent with the server so the panel
  // doesn't need to refetch after every edit.

  const createPersonal = async (content) => {
    try {
      const resp = await api.createAgentCannedResponse({ content })
      const created = resp?.data?.data
      if (created) {
        personal.value = [...personal.value, created]
      }
      return created
    } catch (error) {
      toastError(error)
      return null
    }
  }

  const updatePersonal = async (id, content) => {
    try {
      const resp = await api.updateAgentCannedResponse(id, { content })
      const updated = resp?.data?.data
      if (updated) {
        personal.value = personal.value.map((p) => (p.id === id ? { ...p, ...updated } : p))
        // The personal row's content also backs any existing personal pin,
        // so refresh the resolved pin text in place to keep the "置顶
        // 常用语" group consistent.
        pinned.value = pinned.value.map((p) =>
          p.source === PIN_SOURCE.PERSONAL && p.response_id === id
            ? { ...p, content: updated.content }
            : p
        )
      }
      return updated
    } catch (error) {
      toastError(error)
      return null
    }
  }

  const deletePersonal = async (id) => {
    try {
      await api.deleteAgentCannedResponse(id)
      personal.value = personal.value.filter((p) => p.id !== id)
      // Personal pins for the deleted row are removed by the server via FK
      // cascade; mirror that locally so the pin list does not ghost.
      pinned.value = pinned.value.filter(
        (p) => !(p.source === PIN_SOURCE.PERSONAL && p.response_id === id)
      )
      return true
    } catch (error) {
      toastError(error)
      return false
    }
  }

  const pin = async (source, id) => {
    // Optimistically add to the pin list so the UI updates immediately even
    // before the server response. We need a content snapshot for the local
    // entry; the next `fetch(force)` call will replace it with the
    // authoritative joined view.
    if (source === PIN_SOURCE.PERSONAL) {
      const local = personal.value.find((p) => p.id === id)
      if (local) {
        pinned.value = [
          ...pinned.value,
          {
            source,
            response_id: id,
            content: local.content,
            sort_order: pinned.value.length
          }
        ]
      }
    }
    try {
      await api.pinAgentCannedResponse(source, id)
      await fetch(true)
      return true
    } catch (error) {
      toastError(error)
      // Rollback optimistic update on failure.
      await fetch(true)
      return false
    }
  }

  const unpin = async (source, id) => {
    const previous = pinned.value
    pinned.value = pinned.value.filter(
      (p) => !(p.source === source && p.response_id === id)
    )
    try {
      await api.unpinAgentCannedResponse(source, id)
      return true
    } catch (error) {
      // Restore on failure rather than leaving the user with a vanished
      // pin they have to re-add.
      pinned.value = previous
      toastError(error)
      return false
    }
  }

  return {
    pinned,
    personal,
    globalTree,
    loading,
    loaded,
    fetch,
    createPersonal,
    updatePersonal,
    deletePersonal,
    pin,
    unpin
  }
})
