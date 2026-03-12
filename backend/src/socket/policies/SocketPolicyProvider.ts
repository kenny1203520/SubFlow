import { SocketEventAuthRule } from './SocketPolicyTypes';

export interface SocketPolicyProvider {
    getAuthRules(): Record<string, SocketEventAuthRule>;
}
