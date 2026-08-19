// Deferred promise for App.vue's conversation initialization.
// ChatView awaits this to avoid duplicate initChatConversation calls.
//
// Created at module load time (before any component mounts) so that
// waitForAppInit() always has a promise to await, regardless of whether
// App.vue's onMounted has run yet.
let _initResolve = null
let _initPromise = new Promise((resolve) => {
  _initResolve = resolve
})

export function resolveInit() {
  if (_initResolve) {
    _initResolve()
    _initResolve = null
  }
}

export async function waitForAppInit() {
  if (_initPromise) {
    try { await _initPromise } catch { /* ignore */ }
  }
}
