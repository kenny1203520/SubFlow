import { defineStore } from 'pinia';
import { socket } from '../socket';

export const useAuthStore = defineStore('auth', {
    state: () => ({
        user: null as any,
        isConnected: false
    }),
    getters: {
        isAdmin: (state) => state.user?.isAdmin === true,
        systemRoles: (state) => state.user?.system_roles ?? []
    },
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

