import PocketBase from 'pocketbase'
import { signalClientUpdateRequired, signalMaintenance } from './runtime/maintenance'

export const pb=new PocketBase(window.location.origin)
pb.autoCancellation(false)
pb.beforeSend = (url, options) => {
  const headers = new Headers(options.headers as HeadersInit)
  headers.set('X-SubFlow-Protocol', '1')
  return { url, options: { ...options, headers: Object.fromEntries(headers.entries()) } }
}
pb.afterSend = (response, data) => {
  if (response.status === 503) signalMaintenance()
  if (response.status === 426) signalClientUpdateRequired()
  return data
}

