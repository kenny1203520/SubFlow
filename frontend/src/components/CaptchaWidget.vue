<template>
    <div v-show="provider !== 'none'" class="mb-4 flex justify-center w-full min-h-[78px]">
        <div ref="captchaContainer"></div>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';

declare global {
    interface Window {
        turnstile?: any;
        grecaptcha?: any;
    }
}

const props = defineProps<{
    provider: 'none' | 'turnstile' | 'recaptcha',
    siteKey: string
}>();

const emit = defineEmits<{
    (e: 'verify', token: string): void
}>();

const captchaContainer = ref<HTMLElement | null>(null);
let widgetId: any = null;

const initTurnstile = () => {
    if (!window.turnstile) return;
    if (captchaContainer.value) {
        widgetId = window.turnstile.render(captchaContainer.value, {
            sitekey: props.siteKey,
            callback: (token: string) => {
                emit('verify', token);
            },
            'error-callback': () => {
                emit('verify', '');
            }
        });
    }
};

const initRecaptcha = () => {
    if (!window.grecaptcha) return;

    window.grecaptcha.ready(() => {
        if (captchaContainer.value) {
            widgetId = window.grecaptcha.render(captchaContainer.value, {
                sitekey: props.siteKey,
                callback: (token: string) => {
                    emit('verify', token);
                },
                'expired-callback': () => {
                    emit('verify', '');
                },
                'error-callback': () => {
                    emit('verify', '');
                }
            });
        }
    });
};

const loadScript = (src: string, onLoad: () => void) => {
    if (document.querySelector(`script[src="${src}"]`)) {
        onLoad();
        return;
    }
    const script = document.createElement('script');
    script.src = src;
    script.async = true;
    script.defer = true;
    script.onload = onLoad;
    document.head.appendChild(script);
};

onMounted(() => {
    if (props.provider === 'none' || !props.siteKey) return;

    if (props.provider === 'turnstile') {
        loadScript('https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit', initTurnstile);
    } else if (props.provider === 'recaptcha') {
        loadScript('https://www.google.com/recaptcha/api.js?render=explicit', initRecaptcha);
    }
});

onUnmounted(() => {
    if (props.provider === 'turnstile' && widgetId !== null && window.turnstile) {
        window.turnstile.remove(widgetId);
    } else if (props.provider === 'recaptcha' && widgetId !== null && window.grecaptcha) {
        window.grecaptcha.reset(widgetId);
    }
});

// Expose a reset method in case the parent form needs to manually reset the widget (e.g. on auth failure)
const reset = () => {
    if (props.provider === 'turnstile' && widgetId !== null && window.turnstile) {
        window.turnstile.reset(widgetId);
    } else if (props.provider === 'recaptcha' && widgetId !== null && window.grecaptcha) {
        window.grecaptcha.reset(widgetId);
    }
};

defineExpose({ reset });
</script>
