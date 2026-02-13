import { defineStore } from 'pinia';
import { ref, watch } from 'vue';

export const useLayoutStore = defineStore('layout', () => {
    const isCollapsed = ref(localStorage.getItem('sidebar-collapsed') === 'true');
    const isSidebarOpen = ref(false); // Mobile sidebar state

    watch(isCollapsed, (val) => {
        localStorage.setItem('sidebar-collapsed', val.toString());
    });

    const toggleCollapse = () => {
        isCollapsed.value = !isCollapsed.value;
    };

    const toggleSidebar = () => {
        isSidebarOpen.value = !isSidebarOpen.value;
    };

    const closeSidebar = () => {
        isSidebarOpen.value = false;
    };

    return {
        isCollapsed,
        isSidebarOpen,
        toggleCollapse,
        toggleSidebar,
        closeSidebar
    };
});
