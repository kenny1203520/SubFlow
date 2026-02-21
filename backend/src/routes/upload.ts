import express from 'express';
import multer from 'multer';
import path from 'path';
import { FileService } from '../services/FileService';
import { verifySession, requireAuth, requirePermission } from '../middleware/auth';

const router = express.Router();
const fileService = new FileService();

// Configure storage
const storage = multer.diskStorage({
    destination: (req, file, cb) => {
        const uploadDir = path.join(__dirname, '../../uploads');
        if (!require('fs').existsSync(uploadDir)) {
            require('fs').mkdirSync(uploadDir, { recursive: true });
        }
        cb(null, uploadDir);
    },
    filename: (req, file, cb) => {
        const uniqueSuffix = Date.now() + '-' + Math.round(Math.random() * 1E9);
        cb(null, uniqueSuffix + '-' + file.originalname);
    }
});

const upload = multer({
    storage: storage,
    limits: { fileSize: 5 * 1024 * 1024 }, // 5MB limit
    fileFilter: (req, file, cb) => {
        if (file.mimetype.startsWith('image/')) {
            cb(null, true);
        } else {
            cb(new Error('Only images are allowed'));
        }
    }
});

router.use(verifySession);

// Avatar upload
router.post('/avatar', requireAuth, upload.single('avatar'), requirePermission('user', 'upload', 'avatar'), uploadAvatar);

// Upload file
router.post('/upload', requireAuth, upload.single('file'), requirePermission('user', 'upload', 'file'), uploadFile);

async function uploadAvatar(req: express.Request, res: express.Response) {
    if (!req.file) return res.status(400).json({ error: "No file uploaded" });

    try {
        const userId = res.locals.user!.id;
        const fileRecord = await fileService.recordFile(userId, req.file);

        // Return the public URL (assuming /uploads is served statically or via API)
        const publicUrl = `/uploads/${req.file.filename}`;

        res.json({ status: 'ok', url: publicUrl, file: fileRecord });
    } catch (e: any) {
        res.status(500).json({ error: e.message });
    }
}

async function uploadFile(req: express.Request, res: express.Response) {
    if (!req.file) return res.status(400).json({ error: "No file uploaded" });

    try {
        const userId = res.locals.user!.id;
        const fileRecord = await fileService.recordFile(userId, req.file);
        res.json({ status: 'ok', file: fileRecord });
    } catch (e: any) {
        res.status(500).json({ error: e.message });
    }
}

export default router;
