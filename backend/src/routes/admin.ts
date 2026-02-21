import { Router } from "express";
import { requireAuth, adminOnly } from "../middleware/auth";
import { AdminController } from "../controllers/AdminController";

const router = Router();

// Secure all admin routes
router.use(requireAuth);
router.use(adminOnly);

// --- Dashboard Stats ---
router.get("/stats", AdminController.getStats);

// --- User Management ---
router.get("/users", AdminController.getUsers);
router.put("/users/:id/status", AdminController.updateUserStatus);
router.put("/users/:id/password", AdminController.changeUserPassword);
router.get("/users/:id/sessions", AdminController.getUserSessions);
router.delete("/users/:id/sessions/:sessionId", AdminController.revokeUserSession);

// --- User Role Assignment ---
router.get("/users/:id/roles", AdminController.getUserRoles);
router.post("/users/:id/roles", AdminController.assignUserRole);
router.delete("/users/:id/roles/:roleId", AdminController.removeUserRole);

// --- System Settings ---
router.get("/settings", AdminController.getSettings);
router.put("/settings", AdminController.updateSetting);

// --- System Roles ---
router.get("/roles/system", AdminController.getSystemRoles);

// --- Permissions ---
router.get("/permissions", AdminController.getPermissions);
router.get("/roles/:roleId/permissions", AdminController.getRolePermissions);
router.post("/roles/:roleId/permissions", AdminController.addRolePermission);
router.delete("/roles/:roleId/permissions/:permissionId", AdminController.removeRolePermission);

// --- IP Blocks ---
router.get("/ip-blocks", AdminController.getIpBlocks);
router.post("/ip-blocks", AdminController.blockIp);
router.delete("/ip-blocks/:ip", AdminController.unblockIp);

export default router;
