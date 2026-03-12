import { defineStore } from 'pinia';
import { socket } from '../socket';

export const useAuthStore = defineStore('auth', {
    state: () => ({
        user: null as any,
        isConnected: false
    }),
    getters: {
        systemRoles: (state) => state.user?.system_roles ?? [],
        permissions: (state) => state.user?.permissions ?? [],
        systemPermissions: (state) => state.user?.systemPermissions ?? {},
        hasPermission: (state) => (scope: string, action: string, resource: string) => {
            return state.user?.permissions?.includes(`${scope}:${action}:${resource}`);
        },
        hasSystemPermission: (state) => (action: string, resource: string) => {
            return Boolean(state.user?.systemPermissions?.[`${'system'}:${action}:${resource}`])
                || Boolean(state.user?.systemPermissions?.[action]?.[resource]);
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

