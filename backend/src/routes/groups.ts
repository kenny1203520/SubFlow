import { Router } from "express";
import { requireAuth, requirePermission } from "../middleware/auth";
import { GroupHttpController } from "../controllers/GroupHttpController";

const router = Router();

router.use(requireAuth);

// List user's groups
router.get("/", requirePermission('groups', 'query', 'groups'), GroupHttpController.listUserGroups);

// Create a group
router.post("/", requirePermission('groups', 'create', 'groups'), GroupHttpController.createGroup);

// Get group details
router.get("/:id", requirePermission('groups', 'read', 'groups'), GroupHttpController.getGroupDetails);

export default router;
