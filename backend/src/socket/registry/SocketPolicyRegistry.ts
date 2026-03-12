import { ConventionSocketPolicyProvider } from '../policies/ConventionSocketPolicyProvider';
import { SecuritySocketPolicyProvider } from '../policies/SecuritySocketPolicyProvider';
import { SocketPolicyProvider } from '../policies/SocketPolicyProvider';
import { SocketEventAuthRule } from '../policies/SocketPolicyTypes';

export class SocketPolicyRegistry {
    private readonly providers: SocketPolicyProvider[];
    private cachedRules: Record<string, SocketEventAuthRule> | null = null;

    constructor(providers: SocketPolicyProvider[]) {
        this.providers = providers;
    }

    getAuthRules(): Record<string, SocketEventAuthRule> {
        if (this.cachedRules) {
            return this.cachedRules;
        }

        this.cachedRules = this.providers.reduce<Record<string, SocketEventAuthRule>>((acc, provider) => {
            return {
                ...acc,
                ...provider.getAuthRules()
            };
        }, {});

        return this.cachedRules;
    }

    getAuthRule(eventName: string): SocketEventAuthRule | undefined {
        return this.getAuthRules()[eventName];
    }

    hasAuthRule(eventName: string): boolean {
        return Boolean(this.getAuthRule(eventName));
    }
}

export const socketPolicyRegistry = new SocketPolicyRegistry([
    new ConventionSocketPolicyProvider(),
    new SecuritySocketPolicyProvider()
]);
