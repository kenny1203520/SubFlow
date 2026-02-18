import { AsyncLocalStorage } from 'async_hooks';

export interface AppContext {
    userId: string | null;
}

export const context = new AsyncLocalStorage<AppContext>();

export const runWithContext = <T>(ctx: AppContext, callback: () => T): T => {
    return context.run(ctx, callback);
};

export const getContext = (): AppContext | undefined => {
    return context.getStore();
};
