import { securitySocketEvents } from '../events/SecuritySocketEvents';
import { SocketPolicyProvider } from './SocketPolicyProvider';
import { SocketEventAuthRule } from './SocketPolicyTypes';

export class SecuritySocketPolicyProvider implements SocketPolicyProvider {
    getAuthRules(): Record<string, SocketEventAuthRule> {
        return {
            [securitySocketEvents.GET_SETTINGS]: {
                event: securitySocketEvents.GET_SETTINGS,
                requiredPermission: {
                    scope: 'security',
                    action: 'read',
                    resource: 'security'
                },
                requiresAuthentication: true
            },
            [securitySocketEvents.GENERATE_2FA_SECRET]: {
                event: securitySocketEvents.GENERATE_2FA_SECRET,
                requiredPermission: {
                    scope: 'security',
                    action: 'generate_2fa',
                    resource: 'security'
                },
                requiresAuthentication: true
            },
            [securitySocketEvents.ENABLE_2FA]: {
                event: securitySocketEvents.ENABLE_2FA,
                requiredPermission: {
                    scope: 'security',
                    action: 'enable_2fa',
                    resource: 'security'
                },
                requiresAuthentication: true
            },
            [securitySocketEvents.DISABLE_2FA]: {
                event: securitySocketEvents.DISABLE_2FA,
                requiredPermission: {
                    scope: 'security',
                    action: 'disable_2fa',
                    resource: 'security'
                },
                requiresAuthentication: true
            },
            [securitySocketEvents.REGENERATE_BACKUP_CODES]: {
                event: securitySocketEvents.REGENERATE_BACKUP_CODES,
                requiredPermission: {
                    scope: 'security',
                    action: 'regenerate_backup_codes',
                    resource: 'security'
                },
                requiresAuthentication: true
            },
            [securitySocketEvents.LIST_DEVICES]: {
                event: securitySocketEvents.LIST_DEVICES,
                requiredPermission: {
                    scope: 'security',
                    action: 'read',
                    resource: 'devices'
                },
                requiresAuthentication: true
            },
            [securitySocketEvents.REVOKE_DEVICE]: {
                event: securitySocketEvents.REVOKE_DEVICE,
                requiredPermission: {
                    scope: 'security',
                    action: 'revoke_device',
                    resource: 'devices'
                },
                requiresAuthentication: true
            },
            [securitySocketEvents.SESSIONS]: {
                event: securitySocketEvents.SESSIONS,
                requiredPermission: {
                    scope: 'security',
                    action: 'read',
                    resource: 'sessions'
                },
                requiresAuthentication: true
            },
            [securitySocketEvents.REVOKE_SESSION]: {
                event: securitySocketEvents.REVOKE_SESSION,
                requiredPermission: {
                    scope: 'security',
                    action: 'revoke_session',
                    resource: 'sessions'
                },
                requiresAuthentication: true
            }
        };
    }
}
