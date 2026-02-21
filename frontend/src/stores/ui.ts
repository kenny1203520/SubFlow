import { defineStore } from 'pinia';
import { ref } from 'vue';

export const useUIStore = defineStore('ui', () => {
  const isConfirmOpen = ref(false);
  const confirmMessage = ref('');
  const confirmTitle = ref('');
  const confirmResolve = ref<((value: boolean) => void) | null>(null);

  const isAlertOpen = ref(false);
  const alertMessage = ref('');
  const alertTitle = ref('');
  const alertResolve = ref<(() => void) | null>(null);

  const confirm = (message: string, title: string = 'Confirm') => {
    confirmMessage.value = message;
    confirmTitle.value = title;
    isConfirmOpen.value = true;
    return new Promise<boolean>((resolve) => {
      confirmResolve.value = resolve;
    });
  };

  const alert = (message: string, title: string = 'Notice') => {
    alertMessage.value = message;
    alertTitle.value = title;
    isAlertOpen.value = true;
    return new Promise<void>((resolve) => {
      alertResolve.value = resolve;
    });
  };

  const handleConfirm = (result: boolean) => {
    isConfirmOpen.value = false;
    if (confirmResolve.value) {
        confirmResolve.value(result);
        confirmResolve.value = null;
    }
  };

  const handleAlert = () => {
    isAlertOpen.value = false;
    if (alertResolve.value) {
        alertResolve.value();
        alertResolve.value = null;
    }
  };

  return {
    isConfirmOpen, confirmMessage, confirmTitle, confirm, handleConfirm,
    isAlertOpen, alertMessage, alertTitle, alert, handleAlert
  };
});
