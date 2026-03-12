export interface SocketEventPermission {
    scope: string;
    action: string;
    resource: string;
}

export interface SocketEventAuthRule {
    event: string;
    requiredPermission: SocketEventPermission;
    requiresAuthentication: boolean;
}
