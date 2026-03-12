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
            // console.log(`[MailService] Email to ${to} skipped (ENABLE_EMAIL_SERVICE is not true).`);
            // console.log(`[MailService] Subject: ${subject}`);
            // console.log(`[MailService] Content: ${html}`);
            return true; // Pretend success
        }

        try {
            const info = await this.transporter.sendMail({
                from: process.env.SMTP_FROM || '"SubFlow" <no-reply@subflow.app>',
                to,
                subject,
                html,
            });
            // console.log(`[MailService] Email sent to ${to}: ${info.messageId}`);
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

    static async sendNewDeviceLoginNotification(
        email: string, 
        deviceInfo: {
            deviceName: string;
            ipAddress: string;
            location?: string;
            loginTime: Date;
        }
    ) {
        const manageDevicesLink = `${process.env.FRONTEND_URL || 'http://localhost:5173'}/settings/security/devices`;
        const subject = 'New Device Login Detected - SubFlow';
        const html = `
            <div style="font-family: Arial, sans-serif; padding: 20px; color: #333;">
                <h2 style="color: #EF4444;">🔐 New Device Login Detected</h2>
                <p>We detected a login to your SubFlow account from a new device:</p>
                
                <div style="background-color: #F3F4F6; padding: 15px; border-radius: 8px; margin: 20px 0;">
                    <p style="margin: 5px 0;"><strong>Device:</strong> ${deviceInfo.deviceName}</p>
                    <p style="margin: 5px 0;"><strong>IP Address:</strong> ${deviceInfo.ipAddress}</p>
                    ${deviceInfo.location ? `<p style="margin: 5px 0;"><strong>Location:</strong> ${deviceInfo.location}</p>` : ''}
                    <p style="margin: 5px 0;"><strong>Time:</strong> ${deviceInfo.loginTime.toLocaleString()}</p>
                </div>
                
                <p><strong>Was this you?</strong></p>
                <p>If you recognize this activity, you can ignore this email. If you don't recognize this login:</p>
                <ol>
                    <li>Change your password immediately</li>
                    <li>Review your active devices and revoke any unfamiliar ones</li>
                    <li>Enable two-factor authentication if you haven't already</li>
                </ol>
                
                <a href="${manageDevicesLink}" style="display: inline-block; padding: 10px 20px; background-color: #EF4444; color: white; text-decoration: none; border-radius: 5px; margin-top: 10px;">Manage Devices</a>
                
                <p style="margin-top: 20px; font-size: 12px; color: #666;">For your security, we recommend reviewing your account activity regularly.</p>
            </div>
        `;
        return this.sendMail(email, subject, html);
    }

    static async sendSuspiciousActivityAlert(
        email: string,
        activityInfo: {
            activityType: string;
            details: string;
            ipAddress: string;
            timestamp: Date;
        }
    ) {
        const securityLink = `${process.env.FRONTEND_URL || 'http://localhost:5173'}/settings/security`;
        const subject = '⚠️ Suspicious Activity Detected - SubFlow';
        const html = `
            <div style="font-family: Arial, sans-serif; padding: 20px; color: #333;">
                <h2 style="color: #DC2626;">⚠️ Suspicious Activity Detected</h2>
                <p>We detected suspicious activity on your SubFlow account:</p>
                
                <div style="background-color: #FEE2E2; padding: 15px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #DC2626;">
                    <p style="margin: 5px 0;"><strong>Activity Type:</strong> ${activityInfo.activityType}</p>
                    <p style="margin: 5px 0;"><strong>Details:</strong> ${activityInfo.details}</p>
                    <p style="margin: 5px 0;"><strong>IP Address:</strong> ${activityInfo.ipAddress}</p>
                    <p style="margin: 5px 0;"><strong>Time:</strong> ${activityInfo.timestamp.toLocaleString()}</p>
                </div>
                
                <p><strong style="color: #DC2626;">Immediate Action Required:</strong></p>
                <ol>
                    <li><strong>Change your password immediately</strong></li>
                    <li>Review and revoke unauthorized devices</li>
                    <li>Enable two-factor authentication</li>
                    <li>Check your account activity for any unauthorized changes</li>
                </ol>
                
                <a href="${securityLink}" style="display: inline-block; padding: 10px 20px; background-color: #DC2626; color: white; text-decoration: none; border-radius: 5px; margin-top: 10px;">Secure My Account</a>
                
                <p style="margin-top: 20px; font-size: 12px; color: #666;">If you believe this alert is in error, please contact our support team.</p>
            </div>
        `;
        return this.sendMail(email, subject, html);
    }
}
