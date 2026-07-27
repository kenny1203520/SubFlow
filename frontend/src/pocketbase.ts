import PocketBase from 'pocketbase'

const baseUrl = import.meta.env.VITE_POCKETBASE_URL || window.location.origin

export const pb = new PocketBase(baseUrl)
pb.autoCancellation(false)

export function messageFromError(error: unknown): string {
    if (typeof error === 'object' && error && 'response' in error) {
        const response = (error as { response?: { message?: string; data?: Record<string, { message?: string }> } }).response
        const fieldMessage = response?.data && Object.values(response.data).find((item) => item?.message)?.message
        return fieldMessage || response?.message || '操作失敗，請稍後再試。'
    }
    return '操作失敗，請稍後再試。'
}
