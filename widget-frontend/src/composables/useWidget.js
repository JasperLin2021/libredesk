import { ref, reactive } from 'vue'

const WIDGET_SCRIPT_ID = 'libredesk-widget-script'

/**
 * Composable for managing the LibreDesk chat widget lifecycle.
 * - Loads widget.js via script tag
 * - Hides widget launcher initially; only shows chat after JWT verification
 * - verifyToken() calls POST /api/v1/widget/chat/auth/exchange directly
 */
export function useWidget() {
  const isReady = ref(false)
  const isVisible = ref(false)
  const isAuthenticated = ref(false)
  const checking = ref(false)
  const verifyError = ref(null)
  const unreadCount = ref(0)
  const error = ref(null)
  const userInfo = ref(null)
  const config = reactive({
    baseURL: '',
    inboxID: '',
  })

  // --- Script loading ---
  function loadScript(src) {
    return new Promise((resolve, reject) => {
      const existing = document.getElementById(WIDGET_SCRIPT_ID)
      if (existing) existing.remove()

      const script = document.createElement('script')
      script.id = WIDGET_SCRIPT_ID
      script.src = src
      script.async = true

      script.onload = () => {
        if (window.Libredesk) {
          resolve()
        } else {
          reject(new Error('Widget 脚本已加载但 window.Libredesk 未找到'))
        }
      }
      script.onerror = () => {
        reject(new Error(`加载 Widget 脚本失败: ${src}`))
      }

      document.head.appendChild(script)
    })
  }

  // --- Init Widget ---
  function initWidget(baseURL, inboxID) {
    config.baseURL = baseURL.replace(/\/+$/, '')
    config.inboxID = inboxID
    error.value = null
    isAuthenticated.value = false
    userInfo.value = null

    window.LibredeskSettings = { baseURL: config.baseURL, inboxID }

    const scriptURL = `${config.baseURL}/widget.js`

    loadScript(scriptURL)
      .then(() => {
        const api = window.Libredesk
        if (!api) throw new Error('Widget 加载后不可用')

        api.onShow(() => { isVisible.value = true })
        api.onHide(() => { isVisible.value = false })
        api.onUnreadCountChange((count) => { unreadCount.value = count })

        // Hide the default widget launcher - we only show via our button
        setTimeout(() => {
          if (api && typeof api.hide === 'function') api.hide()
        }, 800)

        isReady.value = true
      })
      .catch((err) => {
        console.error('LibreDesk Widget 初始化失败:', err)
        error.value = err.message
        isReady.value = false
      })
  }

  // --- Token Verification ---
  // Calls POST /api/v1/widget/chat/auth/exchange directly.
  // On success, also calls widget.setUser(jwt) so the iframe picks up the session.
  async function verifyToken(jwt) {
    checking.value = true
    verifyError.value = null

    try {
      const url = `${config.baseURL}/api/v1/widget/chat/auth/exchange`
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Libredesk-Inbox-ID': config.inboxID,
        },
        body: JSON.stringify({ jwt }),
      })

      if (!response.ok) {
        const data = await response.json().catch(() => ({}))
        throw new Error(data.message || '非法用户，请检查 Token')
      }

      const data = await response.json()
      userInfo.value = data.user || null

      // Feed the verified JWT into the widget so its iframe can do its own exchange
      const api = window.Libredesk
      if (api && typeof api.setUser === 'function') {
        api.setUser(jwt)
      }

      isAuthenticated.value = true
      return data
    } catch (err) {
      verifyError.value = err.message
      throw err
    } finally {
      checking.value = false
    }
  }

  // --- Widget Controls ---
  function show() {
    if (window.Libredesk && typeof window.Libredesk.show === 'function') {
      window.Libredesk.show()
    }
  }

  function hide() {
    if (window.Libredesk && typeof window.Libredesk.hide === 'function') {
      window.Libredesk.hide()
    }
  }

  function destroy() {
    if (window.Libredesk && typeof window.Libredesk.destroy === 'function') {
      window.Libredesk.destroy()
    }
    isReady.value = false
    isVisible.value = false
    isAuthenticated.value = false
    verifyError.value = null
    userInfo.value = null
    unreadCount.value = 0
    error.value = null

    const scriptEl = document.getElementById(WIDGET_SCRIPT_ID)
    if (scriptEl) scriptEl.remove()

    delete window.LibredeskSettings
    delete window.Libredesk
  }

  return {
    isReady,
    isVisible,
    isAuthenticated,
    checking,
    verifyError,
    unreadCount,
    error,
    userInfo,
    config,
    initWidget,
    show,
    hide,
    verifyToken,
    destroy,
  }
}
