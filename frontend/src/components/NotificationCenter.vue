<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const isOpen = ref(false);
const notifications = ref<any[]>([]);
const unreadCount = ref(0);

const toggleDropdown = () => {
    isOpen.value = !isOpen.value;
    if (isOpen.value) {
        fetchNotifications();
    }
};

const fetchNotifications = () => {
    socket.emit('notification:list', { page: 1 }, (res: any) => {
        if (res.status === 'ok') {
            notifications.value = res.data.notifications;
            unreadCount.value = res.data.unreadCount;
        }
    });
};

const markRead = (id: string) => {
    socket.emit('notification:mark_read', { ids: [id] }, (res: any) => {
        if (res.status === 'ok') {
            fetchNotifications();
        }
    });
};

const markAllRead = () => {
    socket.emit('notification:mark_all_read', (res: any) => {
        if (res.status === 'ok') {
            fetchNotifications();
        }
    });
};

onMounted(() => {
    fetchNotifications();
    socket.on('notification:new', () => {
        fetchNotifications();
    });
});

onUnmounted(() => {
    socket.off('notification:new');
});
</script>

<template>
    <div class="notification-center">
        <button class="icon-btn glass-card" @click="toggleDropdown">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
            </svg>
            <span v-if="unreadCount > 0" class="badge animate-bounce">{{ unreadCount }}</span>
        </button>

        <Transition name="fade-slide">
            <div v-if="isOpen" class="dropdown glass-panel animate-fade-in">
                <div class="dropdown-header">
                    <h3>{{ t('notifications.title', 'Notifications') }}</h3>
                    <button class="text-btn" @click="markAllRead">{{ t('notifications.mark_all_read', 'Mark all read')
                    }}</button>
                </div>

                <div class="notif-list custom-scrollbar">
                    <div v-if="notifications.length === 0" class="empty-state">
                        {{ t('notifications.empty', 'No notifications') }}
                    </div>
                    <div v-for="notif in notifications" :key="notif.id"
                        :class="['notif-item', { unread: !notif.read_at }]" @click="markRead(notif.id)">
                        <div class="notif-content">
                            <p class="notif-title">{{ notif.title }}</p>
                            <p class="notif-message">{{ notif.message }}</p>
                            <span class="notif-time">{{ new Date(notif.created_at).toLocaleString() }}</span>
                        </div>
                        <div v-if="!notif.read_at" class="unread-dot"></div>
                    </div>
                </div>

                <div class="dropdown-footer">
                    <button class="view-all-btn">{{ t('notifications.view_all', 'View all') }}</button>
                </div>
            </div>
        </Transition>
    </div>
</template>

<style scoped>
.notification-center {
    position: relative;
    z-index: 100;
}

.icon-btn {
    position: relative;
    padding: 0.6rem;
    border-radius: 12px;
    background: white;
    border: none;
    cursor: pointer;
    color: var(--slate-600);
}

.badge {
    position: absolute;
    top: -5px;
    right: -5px;
    background: var(--danger-color);
    color: white;
    font-size: 10px;
    font-weight: 700;
    min-width: 18px;
    height: 18px;
    border-radius: 9px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 2px solid white;
}

.dropdown {
    position: absolute;
    top: calc(100% + 15px);
    right: 0;
    width: 360px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    box-shadow: var(--shadow-lg);
}

.dropdown-header {
    padding: 1.25rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid rgba(0, 0, 0, 0.05);
}

.dropdown-header h3 {
    font-size: 1rem;
    font-weight: 700;
}

.text-btn {
    background: none;
    border: none;
    color: var(--primary-600);
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
}

.notif-list {
    max-height: 400px;
    overflow-y: auto;
}

.notif-item {
    padding: 1rem 1.25rem;
    display: flex;
    gap: 1rem;
    cursor: pointer;
    transition: background 0.2s;
    border-bottom: 1px solid rgba(0, 0, 0, 0.03);
}

.notif-item:hover {
    background: rgba(99, 102, 241, 0.05);
}

.notif-item.unread {
    background: rgba(99, 102, 241, 0.02);
}

.notif-content {
    flex: 1;
}

.notif-title {
    font-weight: 600;
    font-size: 0.875rem;
    margin-bottom: 0.125rem;
}

.notif-message {
    font-size: 0.75rem;
    color: var(--slate-500);
    margin-bottom: 0.5rem;
    line-height: 1.4;
}

.notif-time {
    font-size: 0.7rem;
    color: var(--slate-400);
}

.unread-dot {
    width: 8px;
    height: 8px;
    background: var(--primary-600);
    border-radius: 50%;
    margin-top: 5px;
}

.empty-state {
    padding: 3rem 1.5rem;
    text-align: center;
    color: var(--slate-400);
    font-size: 0.875rem;
}

.dropdown-footer {
    padding: 1rem;
    background: rgba(0, 0, 0, 0.02);
    text-align: center;
}

.view-all-btn {
    width: 100%;
    background: none;
    border: none;
    color: var(--slate-500);
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
}

/* Animations */
.fade-slide-enter-active,
.fade-slide-leave-active {
    transition: all 0.3s ease;
}

.fade-slide-enter-from {
    opacity: 0;
    transform: translateY(-10px);
}

.fade-slide-leave-to {
    opacity: 0;
    transform: translateY(-10px);
}

.custom-scrollbar::-webkit-scrollbar {
    width: 4px;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
    background: var(--slate-200);
    border-radius: 10px;
}
</style>
