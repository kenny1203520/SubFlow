import express from "express";
import { GroupService } from "../services/GroupService";

export class GroupHttpController {
    private static groupService = new GroupService();

    static async listUserGroups(req: express.Request, res: express.Response) {
        const userId = res.locals.user.id;
        try {
            const groups = await this.groupService.listGroups(userId);
            return res.json(groups);
        } catch (error: any) {
            console.error(error);
            return res.status(500).send("Server Error");
        }
    }

    static async createGroup(req: express.Request, res: express.Response) {
        const userId = res.locals.user.id;
        const { name } = req.body;

        if (typeof name !== "string" || name.trim().length === 0) {
            return res.status(400).send("Invalid group name");
        }

        try {
            const newGroup = await this.groupService.createGroup(userId, { name: name.trim() });
            return res.status(201).json(newGroup);
        } catch (error: any) {
            console.error(error);
            return res.status(500).send("Server Error");
        }
    }

    static async getGroupDetails(req: express.Request, res: express.Response) {
        const userId = res.locals.user.id;
        const { id } = req.params;
        const groupId = Array.isArray(id) ? id[0] : id;

        try {
            const detail = await this.groupService.getGroupDetail(userId, groupId);
            return res.json(detail);
        } catch (error: any) {
            console.error(error);
            return res.status(500).send("Server Error");
        }
    }
}
