// The isolated appliance edition deliberately trusts its private network
// boundary. Keep the check explicit so no-auth mode never enters the legacy
// Google fallback in useAuth.
export function isNoAuthProvider(provider) {
  return provider === 'disabled'
}
