import { Router } from "express";
import { requireAuth, adminOnly } from "../middleware/auth";
import { AdminController } from "../controllers/AdminController";

const router = Router();

// Secure all admin routes
router.use(requireAuth);
router.use(adminOnly);

// --- User Management ---
router.get("/users", AdminController.getUsers);
router.put("/users/:id/status", AdminController.updateUserStatus);
router.get("/users/:id/sessions", AdminController.getUserSessions);
router.delete("/users/:id/sessions/:sessionId", AdminController.revokeUserSession);
// router.post("/users/:id/reset-password", AdminController.resetUserPassword);

// --- System Settings ---
router.get("/settings", AdminController.getSettings);
router.put("/settings", AdminController.updateSetting);

// --- Roles ---
router.get("/roles/system", AdminController.getSystemRoles);

export default router;
