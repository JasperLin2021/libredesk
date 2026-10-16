import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'

/**
 * Shared state and actions for the "online blacklist" feature.
 *
 * Every caller (conversation sidebar, customer list, operation log) talks to
 * the same three endpoints, so the fetching / error handling / toasting lives
 * here once instead of being duplicated per view.
 */
export function useBlacklist () {
  const { t } = useI18n()
  const emitter = useEmitter()

  // Whether the contact is currently on the online blacklist.
  const blocked = ref(false)
  // Reason recorded when the contact was blacklisted.
  const reason = ref('')
  const loading = ref(false)
  const submitting = ref(false)

  // Guards against out of order responses when the contact changes quickly.
  let requestId = 0

  const toast = (description, variant) => {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description, variant })
  }

  const toastError = (err, fallbackKey) => {
    toast(err ? handleHTTPError(err).message : t(fallbackKey), 'destructive')
  }

  const fetchStatus = async (contactId) => {
    if (!contactId) {
      blocked.value = false
      reason.value = ''
      return false
    }
    const current = ++requestId
    loading.value = true
    try {
      const resp = await api.getContactBlacklist(contactId)
      if (current !== requestId) return false
      const data = resp?.data?.data || {}
      blocked.value = !!data.blocked
      reason.value = data.reason || ''
      return true
    } catch (err) {
      if (current !== requestId) return false
      toastError(err, 'blacklist.statusError')
      return false
    } finally {
      if (current === requestId) loading.value = false
    }
  }

  const add = async (contactId, text) => {
    submitting.value = true
    try {
      const resp = await api.addContactBlacklist(contactId, { reason: text })
      const data = resp?.data?.data || {}
      blocked.value = !!data.blocked
      reason.value = data.reason || text || ''
      toast(t('blacklist.addedSuccessfully'))
      return true
    } catch (err) {
      toastError(err, 'blacklist.saveError')
      return false
    } finally {
      submitting.value = false
    }
  }

  const remove = async (contactId) => {
    submitting.value = true
    try {
      const resp = await api.removeContactBlacklist(contactId)
      const data = resp?.data?.data || {}
      blocked.value = !!data.blocked
      reason.value = data.blocked ? data.reason || '' : ''
      toast(t('blacklist.removedSuccessfully'))
      return true
    } catch (err) {
      toastError(err, 'blacklist.saveError')
      return false
    } finally {
      submitting.value = false
    }
  }

  return {
    blocked,
    reason,
    loading,
    submitting,
    fetchStatus,
    add,
    remove
  }
}

export default useBlacklist
