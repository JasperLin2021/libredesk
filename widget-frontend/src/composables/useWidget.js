import { ref, reactive, shallowRef } from 'vue'

const WIDGET_SCRIPT_ID = 'libredesk-widget-script'

/**
 * Composable for managing the LibreDesk chat widget lifecycle.
 * Dynamically loads widget.js, initializes with config, and exposes
 * reactive state + API proxy methods.
 */
export function useWidget() {
  const isReady = ref(false)
  const isVisible = ref(false)
  const unreadCount = ref(0)
  const error = ref(null)
  const config = reactive({
    baseURL: '',
    inboxID: '',
  })

  let widgetInstance = null

  function loadScript(src) {
    return new Promise((resolve, reject) => {
      const existing = document.getElementById(WIDGET_SCRIPT_ID)
      if (existing) {
        existing.remove()
      }

      const script = document.createElement('script')
      script.id = WIDGET_SCRIPT_ID
      script.src = src
      script.async = true

      script.onload = () => {
        if (window.Libredesk && typeof window.initLibredesk === 'function') {
          resolve()
        } else {
          reject(new Error('Widget script loaded but Libredesk/initLibredesk not found'))
        }
      }
      script.onerror = () => {
        reject(new Error(`Failed to load widget script from: ${src}`))
      }

      document.head.appendChild(script)
    })
  }

  function initWidget(baseURL, inboxID) {
    config.baseURL = baseURL
    config.inboxID = inboxID
    error.value = null

    const scriptURL = `${baseURL}/widget.js`

    loadScript(scriptURL)
      .then(() => {
        const instance = window.initLibredesk({ baseURL, inboxID })
        widgetInstance = instance

        instance.onShow(() => {
          isVisible.value = true
        })

        instance.onHide(() => {
          isVisible.value = false
        })

        instance.onUnreadCountChange((count) => {
          unreadCount.value = count
        })

        isReady.value = true
      })
      .catch((err) => {
        console.error('LibreDesk Widget initialization failed:', err)
        error.value = err.message
        isReady.value = false
      })
  }

  function show() {
    if (widgetInstance && typeof widgetInstance.show === 'function') {
      widgetInstance.show()
    }
  }

  function hide() {
    if (widgetInstance && typeof widgetInstance.hide === 'function') {
      widgetInstance.hide()
    }
  }

  function setUser(jwt) {
    if (widgetInstance && typeof widgetInstance.setUser === 'function') {
      widgetInstance.setUser(jwt)
    }
  }

  function logout() {
    if (widgetInstance && typeof widgetInstance.logout === 'function') {
      widgetInstance.logout()
    }
  }

  function destroy() {
    if (widgetInstance && typeof widgetInstance.destroy === 'function') {
      widgetInstance.destroy()
    }
    widgetInstance = null
    isReady.value = false
    isVisible.value = false
    unreadCount.value = 0
    error.value = null

    const scriptEl = document.getElementById(WIDGET_SCRIPT_ID)
    if (scriptEl) {
      scriptEl.remove()
    }
  }

  return {
    isReady,
    isVisible,
    unreadCount,
    error,
    config,
    initWidget,
    show,
    hide,
    setUser,
    logout,
    destroy,
  }
}
