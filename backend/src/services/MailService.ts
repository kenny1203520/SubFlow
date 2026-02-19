import dotenv from 'dotenv';
import nodemailer from 'nodemailer';

dotenv.config();

export class MailService {
    private static transporter = nodemailer.createTransport({
        host: process.env.SMTP_HOST,
        port: parseInt(process.env.SMTP_PORT || '587'),
        secure: process.env.SMTP_SECURE === 'true', // true for 465, false for other ports
        auth: {
            user: process.env.SMTP_USER,
            pass: process.env.SMTP_PASS,
        },
    });

    private static async sendMail(to: string, subject: string, html: string): Promise<boolean> {
        if (process.env.ENABLE_EMAIL_SERVICE !== 'true') {
            console.log(`[MailService] Email to ${to} skipped (ENABLE_EMAIL_SERVICE is not true).`);
            console.log(`[MailService] Subject: ${subject}`);
            console.log(`[MailService] Content: ${html}`);
            return true; // Pretend success
        }

        try {
            const info = await this.transporter.sendMail({
                from: process.env.SMTP_FROM || '"SubFlow" <no-reply@subflow.app>',
                to,
                subject,
                html,
            });
            console.log(`[MailService] Email sent to ${to}: ${info.messageId}`);
            return true;
        } catch (error) {
            console.error(`[MailService] Failed to send email to ${to}:`, error);
            return false;
        }
    }

    static async sendPasswordResetEmail(email: string, token: string) {
        const resetLink = `${process.env.FRONTEND_URL || 'http://localhost:5173'}/auth/reset-password/${token}`;
        const subject = 'Reset Your Password - SubFlow';
        const html = `
            <div style="font-family: Arial, sans-serif; padding: 20px; color: #333;">
                <h2>Reset Your Password</h2>
                <p>You requested a password reset. Click the button below to reset your password:</p>
                <a href="${resetLink}" style="display: inline-block; padding: 10px 20px; background-color: #4F46E5; color: white; text-decoration: none; border-radius: 5px; margin-top: 10px;">Reset Password</a>
                <p style="margin-top: 20px; font-size: 12px; color: #666;">If you didn't request this, you can ignore this email.</p>
                <p style="font-size: 10px; color: #999;">Link expires in 1 hour.</p>
            </div>
        `;
        return this.sendMail(email, subject, html);
    }

    static async sendVerificationEmail(email: string, token: string) {
        const verificationLink = `${process.env.FRONTEND_URL || 'http://localhost:5173'}/auth/verify-email/${token}`;
        const subject = 'Verify Your Email - SubFlow';
        const html = `
            <div style="font-family: Arial, sans-serif; padding: 20px; color: #333;">
                <h2>Verify Your Email</h2>
                <p>Welcome to SubFlow! Please verify your email address by clicking the button below:</p>
                <a href="${verificationLink}" style="display: inline-block; padding: 10px 20px; background-color: #10B981; color: white; text-decoration: none; border-radius: 5px; margin-top: 10px;">Verify Email</a>
                <p style="margin-top: 20px; font-size: 12px; color: #666;">If you didn't sign up for SubFlow, you can ignore this email.</p>
            </div>
        `;
        return this.sendMail(email, subject, html);
    }
}
