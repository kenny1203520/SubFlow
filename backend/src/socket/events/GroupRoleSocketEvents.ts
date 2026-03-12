export class GroupRoleSocketEvents {
    readonly LIST = 'group:role:list';
    readonly QUANTITY_LIMIT = 'group:role:quantity_limit';
    readonly USER_MAX_ROLE_LEVEL = 'group:user:max_role_level';
    readonly GET_PERMISSIONS = 'group:role:get_permissions';
    readonly CREATE = 'group:role:create';
    readonly UPDATE = 'group:role:update';
    readonly UPDATE_LEVEL = 'group:role:update_level';
    readonly DELETE = 'group:role:delete';
    readonly ASSIGN = 'group:role:assign';
    readonly REMOVE = 'group:role:remove';
    readonly GRANT_PERMISSION = 'group:role:grant_permission';
    readonly REVOKE_PERMISSION = 'group:role:revoke_permission';
    readonly MEMBER_GRANT_PERMISSION = 'group:member:grant_permission';
    readonly MEMBER_REVOKE_PERMISSION = 'group:member:revoke_permission';
    readonly MEMBER_LIST_DIRECT_PERMISSIONS = 'group:member:list_direct_permissions';
    readonly LIST_ALL_PERMISSIONS = 'group:permissions:list';
    readonly TRANSFER_OWNERSHIP = 'group:ownership:transfer';

    all(): string[] {
        return [
            this.LIST,
            this.QUANTITY_LIMIT,
            this.USER_MAX_ROLE_LEVEL,
            this.GET_PERMISSIONS,
            this.CREATE,
            this.UPDATE,
            this.UPDATE_LEVEL,
            this.DELETE,
            this.ASSIGN,
            this.REMOVE,
            this.GRANT_PERMISSION,
            this.REVOKE_PERMISSION,
            this.MEMBER_GRANT_PERMISSION,
            this.MEMBER_REVOKE_PERMISSION,
            this.MEMBER_LIST_DIRECT_PERMISSIONS,
            this.LIST_ALL_PERMISSIONS,
            this.TRANSFER_OWNERSHIP
        ];
    }
}

export const groupRoleSocketEvents = new GroupRoleSocketEvents();
