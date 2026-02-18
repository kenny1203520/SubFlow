import { pool } from '../db';

export interface ServiceRow {
    id: string;
    name: string;
    domain: string | null;
    icon_url: string | null;
    is_system: boolean;
    created_by: string;
    created_at: Date;
}

export class ServiceService {

    async searchServices(query: string): Promise<ServiceRow[]> {
        const result = await pool.query(
            "SELECT * FROM services WHERE name ILIKE $1 OR domain ILIKE $1 LIMIT 10",
            [`%${query}%`]
        );
        return result.rows;
    }

    async createService(userId: string, payload: { name: string, domain?: string, icon_url?: string }): Promise<ServiceRow> {
        // Auto-fetch icon if domain is provided and icon is not
        let iconUrl = payload.icon_url || '';
        if (payload.domain && !iconUrl) {
            try {
                const domain = new URL(payload.domain.startsWith('http') ? payload.domain : `https://${payload.domain}`).hostname;
                iconUrl = `https://www.google.com/s2/favicons?domain=${domain}&sz=64`;
            } catch (e) { }
        }

        const res = await pool.query(
            "INSERT INTO services (name, domain, icon_url, created_by) VALUES ($1, $2, $3, $4) RETURNING *",
            [payload.name, payload.domain || null, iconUrl, userId]
        );
        return res.rows[0];
    }

    async listServices(): Promise<ServiceRow[]> {
        const res = await pool.query("SELECT * FROM services ORDER BY name ASC");
        return res.rows;
    }

    async updateService(id: string, payload: { name?: string, domain?: string, icon_url?: string }): Promise<ServiceRow> {
        // Build dynamic query
        const fields: string[] = [];
        const values: any[] = [];
        let idx = 1;

        if (payload.name !== undefined) {
            fields.push(`name = $${idx++}`);
            values.push(payload.name);
        }
        if (payload.domain !== undefined) {
            fields.push(`domain = $${idx++}`);
            values.push(payload.domain);
        }
        if (payload.icon_url !== undefined) {
            fields.push(`icon_url = $${idx++}`);
            values.push(payload.icon_url);
        }

        if (fields.length === 0) throw new Error("No fields to update");

        values.push(id);
        const res = await pool.query(
            `UPDATE services SET ${fields.join(', ')}, updated_at = NOW() WHERE id = $${idx} RETURNING *`,
            values
        );

        if (res.rows.length === 0) throw new Error("Service not found");
        return res.rows[0];
    }

    async deleteService(id: string) {
        // Check if used in groups? For now just allow delete or restrict. 
        // Ideally should check constraints or cascade. 
        // Let's assume soft delete or check existance.
        const check = await pool.query("SELECT 1 FROM groups WHERE service_id = $1", [id]);
        if (check.rows.length > 0) {
            throw new Error("Cannot delete service that is in use by groups");
        }
        await pool.query("DELETE FROM services WHERE id = $1", [id]);
    }
}
