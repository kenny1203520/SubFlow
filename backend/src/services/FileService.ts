import { FileRepository } from '../repositories/FileRepository';
import fs from 'fs';
import path from 'path';

export class FileService {
    private fileRepo = new FileRepository();

    async recordFile(userId: string, file: Express.Multer.File) {
        return await this.fileRepo.create({
            file_name: file.originalname,
            file_path: file.path,
            file_size: file.size,
            file_type: file.mimetype,
            uploaded_by: userId
        });
    }

    async getFile(fileId: string) {
        return await this.fileRepo.findById(fileId);
    }

    async deleteFile(fileId: string, userId: string) {
        const file = await this.fileRepo.findById(fileId);
        if (!file) throw new Error("File not found");
        // Optional: Check ownership or admin rights

        // Delete from disk
        // Note: In real prod, this might be S3. Here it's local FS.
        if (fs.existsSync(file.file_path)) {
            fs.unlinkSync(file.file_path);
        }

        await this.fileRepo.delete(fileId);
    }
}
