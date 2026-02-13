import { defineStore } from 'pinia';
import { socket } from '../socket';

export const useAuthStore = defineStore('auth', {
    state: () => ({
        user: null as any,
        isConnected: false
    }),
    actions: {
        setUser(user: any) {
            this.user = user;
            if (user && !this.isConnected) {
                socket.connect();
                this.isConnected = true;
            }
        },
        clearUser() {
            this.user = null;
            socket.disconnect();
            this.isConnected = false;
        }
    }
});
