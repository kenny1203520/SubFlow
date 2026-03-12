export class GroupSocketEvents {
    readonly CREATE = 'group:create';
    readonly LIST = 'group:list';
    readonly GET = 'group:get';
    readonly GET_OVERVIEW = 'group:get_overview';
    readonly ADD_MEMBER = 'group:add_member';
    readonly BIND_MEMBER = 'group:bind_member';
    readonly DELETE = 'group:delete';
    readonly LEAVE = 'group:leave';
    readonly ACCEPT_INVITE = 'group:accept_invite';
    readonly REJECT_INVITE = 'group:reject_invite';
    readonly BIND_MEMBER_INVITE = 'group:bind_member_invite';
    readonly UPDATE = 'group:update';
    readonly UPDATE_MEMBER_ROLE = 'group:update_member_role';
    readonly ASSIGN_DYNAMIC_ROLE = 'group:assign_dynamic_role';
    readonly REMOVE_DYNAMIC_ROLE = 'group:remove_dynamic_role';
    readonly REMOVE_MEMBER = 'group:remove_member';
    readonly CANCEL_INVITE = 'group:cancel_invite';
    readonly LIST_ROLES = 'group:list_roles';
    readonly CREATE_ROLE = 'group:create_role';

    all(): string[] {
        return [
            this.CREATE,
            this.LIST,
            this.GET,
            this.GET_OVERVIEW,
            this.ADD_MEMBER,
            this.BIND_MEMBER,
            this.DELETE,
            this.LEAVE,
            this.ACCEPT_INVITE,
            this.REJECT_INVITE,
            this.BIND_MEMBER_INVITE,
            this.UPDATE,
            this.UPDATE_MEMBER_ROLE,
            this.ASSIGN_DYNAMIC_ROLE,
            this.REMOVE_DYNAMIC_ROLE,
            this.REMOVE_MEMBER,
            this.CANCEL_INVITE,
            this.LIST_ROLES,
            this.CREATE_ROLE
        ];
    }
}

export const groupSocketEvents = new GroupSocketEvents();
