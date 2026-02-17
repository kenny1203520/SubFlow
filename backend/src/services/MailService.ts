import dotenv from 'dotenv';
dotenv.config();

export class MailService {
    static async sendPasswordResetEmail(email: string, token: string) {
        const resetLink = `${process.env.FRONTEND_URL || 'http://localhost:5173'}/auth/reset-password/${token}`;

        console.log('--- [MAIL SIMULATION] ---');
        console.log(`To: ${email}`);
        console.log(`Subject: Reset Your Password`);
        console.log(`Message: Click the following link to reset your password:`);
        console.log(resetLink);
        console.log('-------------------------');

        // In production, integrate with SendGrid, Mailtrap, or AWS SES
        return true;
    }

    static async sendVerificationEmail(email: string, token: string) {
        const verificationLink = `${process.env.FRONTEND_URL || 'http://localhost:5173'}/auth/verify-email/${token}`;

        console.log('--- [MAIL SIMULATION] ---');
        console.log(`To: ${email}`);
        console.log(`Subject: Verify Your Email`);
        console.log(`Message: Click the following link to verify your email:`);
        console.log(verificationLink);
        console.log('-------------------------');

        return true;
    }
}
