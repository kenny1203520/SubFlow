import PocketBase from 'pocketbase'
export const pb=new PocketBase(window.location.origin)
pb.autoCancellation(false)

