import { socketEventRegistry } from '../registry/SocketEventRegistry';
import { SocketPolicyProvider } from './SocketPolicyProvider';
import { SocketEventAuthRule } from './SocketPolicyTypes';

export class ConventionSocketPolicyProvider implements SocketPolicyProvider {
    getAuthRules(): Record<string, SocketEventAuthRule> {
        const rules: Record<string, SocketEventAuthRule> = {};

        for (const eventName of socketEventRegistry.getAll()) {
            rules[eventName] = this.buildRule(eventName);
        }

        return rules;
    }

    private buildRule(eventName: string): SocketEventAuthRule {
        if (eventName === 'ping') {
            return {
                event: eventName,
                requiredPermission: {
                    scope: 'system',
                    action: 'ping',
                    resource: 'system'
                },
                requiresAuthentication: false
            };
        }

        const parts = eventName.split(':');
        if (parts.length === 1) {
            return {
                event: eventName,
                requiredPermission: {
                    scope: parts[0],
                    action: 'execute',
                    resource: parts[0]
                },
                requiresAuthentication: true
            };
        }

        if (parts.length === 2) {
            return {
                event: eventName,
                requiredPermission: {
                    scope: parts[0],
                    action: parts[1],
                    resource: parts[0]
                },
                requiresAuthentication: true
            };
        }

        return {
            event: eventName,
            requiredPermission: {
                scope: `${parts[0]}:${parts[1]}`,
                action: parts.slice(2).join('_'),
                resource: parts[1]
            },
            requiresAuthentication: true
        };
    }
}
