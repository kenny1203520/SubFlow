import { BaseRepository } from "./BaseRepository";

export interface SystemSettingsRow {
    key: string;
    value: any;
    description?: string;
    updated_at?: Date;
    updated_by?: string;
}

export class SystemSettingRepository extends BaseRepository {

    async getSetting(key: string): Promise<SystemSettingsRow | null> {
        const res = await this.query("SELECT * FROM system_settings WHERE key = $1", [key]);
        return res.rows[0] || null;
    }

    async setSetting(key: string, value: any, updatedBy: string, description?: string): Promise<void> {
        await this.query(`
            INSERT INTO system_settings (key, value, description, updated_by) 
            VALUES ($1, $2, $3, $4) 
            ON CONFLICT (key)
            DO
                UPDATE 
                SET value = EXCLUDED.value, description = EXCLUDED.description, updated_by = EXCLUDED.updated_by
            `, [key, value, description, updatedBy]
        );
    }

    async deleteSetting(key: string): Promise<void> {
        await this.query("DELETE FROM system_settings WHERE key = $1", [key]);
    }

}