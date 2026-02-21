import { Router } from "express";
import { requireAuth, requirePermission } from "../middleware/auth";
import { AdminController } from "../controllers/AdminController";

const router = Router();

// Secure all admin routes with general authentication
router.use(requireAuth);

// Base admin access (dashboard stats etc might just need adminOnly or a basic read permission)
// For now, we keep adminOnly for base access, but use requirePermission for granular actions.
router.get("/stats", requirePermission('system', 'read', 'stats'), AdminController.getStats);

// --- User Management ---
router.get("/users", requirePermission('system', 'read', 'users'), AdminController.getUsers);
router.put("/users/:id/status", requirePermission('system', 'update', 'users'), AdminController.updateUserStatus);
router.put("/users/:id/password", requirePermission('system', 'update', 'users'), AdminController.changeUserPassword);
router.get("/users/:id/sessions", requirePermission('system', 'read', 'sessions'), AdminController.getUserSessions);
router.delete("/users/:id/sessions/:sessionId", requirePermission('system', 'delete', 'sessions'), AdminController.revokeUserSession);

// --- User Role Assignment ---
router.get("/users/:id/roles", requirePermission('system', 'read', 'user_roles'), AdminController.getUserRoles);
router.post("/users/:id/roles", requirePermission('system', 'manage', 'user_roles'), AdminController.assignUserRole);
router.delete("/users/:id/roles/:roleId", requirePermission('system', 'manage', 'user_roles'), AdminController.removeUserRole);

// --- User Direct Permissions ---
router.get("/users/:id/permissions", requirePermission('system', 'read', 'permissions_user'), AdminController.getUserPermissions);
router.post("/users/:id/permissions", requirePermission('system', 'manage', 'permissions_user'), AdminController.addUserPermission);
router.delete("/users/:id/permissions/:permissionId", requirePermission('system', 'manage', 'permissions_user'), AdminController.removeUserPermission);

// --- System Settings ---
router.get("/settings", requirePermission('system', 'read', 'settings'), AdminController.getSettings);
router.put("/settings", requirePermission('system', 'update', 'settings'), AdminController.updateSetting);

// --- System Roles ---
router.get("/roles/system", requirePermission('system', 'read', 'roles'), AdminController.getSystemRoles);
router.post("/roles/system", requirePermission('system', 'manage', 'roles'), AdminController.createRole);
router.put("/roles/system/:id", requirePermission('system', 'manage', 'roles'), AdminController.updateRole);
router.delete("/roles/system/:id", requirePermission('system', 'manage', 'roles'), AdminController.deleteRole);

// --- Permissions ---
router.get("/permissions", requirePermission('system', 'read', 'permissions'), AdminController.getPermissions);
router.get("/roles/:roleId/permissions", requirePermission('system', 'read', 'roles'), AdminController.getRolePermissions);
router.post("/roles/:roleId/permissions", requirePermission('system', 'manage', 'roles'), AdminController.addRolePermission);
router.delete("/roles/:roleId/permissions/:permissionId", requirePermission('system', 'manage', 'roles'), AdminController.removeRolePermission);

// --- Activity Logs ---
router.get("/logs", requirePermission('system', 'read', 'logs'), AdminController.getActivityLogs);

// --- IP Blocks ---
router.get("/ip-blocks", requirePermission('system', 'read', 'ip_blocks'), AdminController.getIpBlocks);
router.post("/ip-blocks", requirePermission('system', 'create', 'ip_blocks'), AdminController.blockIp);
router.delete("/ip-blocks/:ip", requirePermission('system', 'delete', 'ip_blocks'), AdminController.unblockIp);

export default router;
