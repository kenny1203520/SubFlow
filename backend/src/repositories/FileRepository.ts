import { BaseRepository } from './BaseRepository';

export interface FileRow {
    id: string;
    file_name: string;
    file_path: string;
    file_size: number;
    file_type: string;
    uploaded_by: string;
    created_at: Date;
}

export class FileRepository extends BaseRepository {
    async create(data: Partial<FileRow>): Promise<FileRow> {
        const res = await this.query(
            "INSERT INTO files (file_name, file_path, file_size, file_type, uploaded_by) VALUES ($1, $2, $3, $4, $5) RETURNING *",
            [data.file_name, data.file_path, data.file_size, data.file_type, data.uploaded_by]
        );
        return res.rows[0];
    }

    async findById(id: string): Promise<FileRow | null> {
        const res = await this.query("SELECT * FROM files WHERE id = $1", [id]);
        return res.rows[0] || null;
    }

    async delete(id: string): Promise<void> {
        await this.query("DELETE FROM files WHERE id = $1", [id]);
    }
}
