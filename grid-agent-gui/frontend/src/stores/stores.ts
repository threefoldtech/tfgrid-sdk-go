import { writable } from 'svelte/store';

export const themeStore = writable('dark');
export const settingsStore = writable({
    mnemonics: '',
    network: 'main',
    geminiApiKey: '',
    theme: 'dark',
    isConfigured: false
});
export const messagesStore = writable([]);
