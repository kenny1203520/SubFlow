import { defineStore } from 'pinia';
import { socket } from '../socket';

export const useAuthStore = defineStore('auth', {
    state: () => ({
        user: null as any,
        isConnected: false
    }),
    getters: {
        isAdmin: (state) => state.user?.isAdmin === true,
        systemRoles: (state) => state.user?.system_roles ?? [],
        permissions: (state) => state.user?.permissions ?? [],
        hasPermission: (state) => (scope: string, action: string, resource: string) => {
            return state.user?.permissions?.includes(`${scope}:${action}:${resource}`);
        }
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

