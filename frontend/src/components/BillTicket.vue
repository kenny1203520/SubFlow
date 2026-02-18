<script setup lang="ts">
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const props = defineProps<{
    group: any;
    member: any;
    split: any;
    show: boolean;
}>();

const emit = defineEmits(['close']);

const formatDate = (date: string) => {
    return new Date(date).toLocaleDateString();
};
</script>

<template>
    <div v-if="show" class="modal-overlay" @click.self="$emit('close')">
        <div class="ticket">
            <div class="ticket-header">
                <div class="logo">
                    <img src="/favicon.svg" style="width: 24px; height: 24px;" alt="S" />
                </div>
                <div class="brand">SubFlow</div>
                <div class="ticket-type">{{ t('groups.nonMember') }}</div>
            </div>

            <div class="ticket-body">
                <div class="service-info">
                    <h3>{{ group.service_name || group.name }}</h3>
                    <p>{{ group.plan_name || '' }}</p>
                </div>

                <div class="divider"></div>

                <div class="details">
                    <div class="detail-row">
                        <span class="label">{{ t('common.fields.member') }}</span>
                        <span class="value">{{ member?.username || member?.temp_name }}</span>
                    </div>
                    <div class="detail-row">
                        <span class="label">{{ t('common.fields.date') }}</span>
                        <span class="value">{{ formatDate(split.date || new Date().toISOString()) }}</span>
                    </div>
                    <div class="detail-row">
                        <span class="label">{{ t('groups.billingCycle') }}</span>
                        <span class="value">{{ t(`common.cycles.${group.billing_cycle}`) }}</span>
                    </div>
                </div>

                <div class="amount-section">
                    <span class="label">{{ t('groups.amount') }}</span>
                    <span class="amount">{{ group.currency }} {{ split.amount_owed }}</span>
                </div>
            </div>

            <div class="ticket-footer">
                <div class="barcode"></div>
                <p class="footer-note">{{ t('groups.createdOn', { date: formatDate(new Date().toISOString()) }) }}</p>
                <button @click="$emit('close')" class="btn-close">{{ t('common.actions.close') }}</button>
            </div>
        </div>
    </div>
</template>

<style scoped>
.modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.7);
    backdrop-filter: blur(8px);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 2000;
}

.ticket {
    background: white;
    width: 350px;
    border-radius: 16px;
    overflow: hidden;
    color: #1a1a1a;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
    position: relative;
}

.ticket::before,
.ticket::after {
    content: '';
    position: absolute;
    width: 30px;
    height: 30px;
    background: rgba(0, 0, 0, 0.7);
    border-radius: 50%;
    top: 50%;
    margin-top: -15px;
    z-index: 10;
}

.ticket::before {
    left: -15px;
}

.ticket::after {
    right: -15px;
}

.ticket-header {
    background: var(--primary-600);
    color: white;
    padding: 1.5rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
}

.logo {
    width: 28px;
    height: 28px;
    background: white;
    color: var(--primary-600);
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 800;
}

.brand {
    font-weight: 700;
    font-size: 1.125rem;
}

.ticket-type {
    margin-left: auto;
    font-size: 0.65rem;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    opacity: 0.8;
}

.ticket-body {
    padding: 2rem 1.5rem;
    background: linear-gradient(to bottom, #ffffff, #f9fafb);
}

.service-info {
    text-align: center;
    margin-bottom: 1.5rem;
}

.service-info h3 {
    font-size: 1.5rem;
    font-weight: 800;
    margin: 0;
    color: #111827;
}

.service-info p {
    color: var(--text-muted);
    font-size: 0.875rem;
    margin-top: 0.25rem;
}

.divider {
    border-top: 2px dashed #e5e7eb;
    margin: 1.5rem 0;
    position: relative;
}

.details {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.detail-row {
    display: flex;
    justify-content: space-between;
    font-size: 0.875rem;
}

.detail-row .label {
    color: var(--text-muted);
}

.detail-row .value {
    font-weight: 600;
}

.amount-section {
    margin-top: 2rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
}

.amount-section .label {
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--text-muted);
}

.amount-section .amount {
    font-size: 2.5rem;
    font-weight: 900;
    color: var(--primary-700);
}

.ticket-footer {
    padding: 1.5rem;
    background: #f3f4f6;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
}

.barcode {
    width: 100%;
    height: 40px;
    background: repeating-linear-gradient(to right,
            #000,
            #000 2px,
            #fff 2px,
            #fff 4px);
    opacity: 0.2;
}

.footer-note {
    font-size: 0.7rem;
    color: var(--text-muted);
}

.btn-close {
    width: 100%;
    padding: 0.75rem;
    background: white;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.2s;
}

.btn-close:hover {
    background: #f9fafb;
}
</style>
