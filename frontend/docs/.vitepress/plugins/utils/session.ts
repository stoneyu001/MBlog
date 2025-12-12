/**
 * 会话管理工具
 */

const isBrowser = typeof window !== 'undefined';

export interface SessionData {
    id: string;
    fingerprint: string;
    created: number;
    last_activity: number;
}

/**
 * 生成会话ID
 */
export function generateSessionId(): string {
    return Date.now().toString(36) + Math.random().toString(36).substring(2);
}

/**
 * 从 sessionStorage 获取会话数据
 */
export function getSessionData(key: string): SessionData | null {
    if (!isBrowser) return null;

    try {
        const data = sessionStorage.getItem(key);
        return data ? JSON.parse(data) : null;
    } catch (error) {
        console.error('[Session] Failed to get session data:', error);
        return null;
    }
}

/**
 * 保存会话数据到 sessionStorage
 */
export function saveSessionData(key: string, data: SessionData): void {
    if (!isBrowser) return;

    try {
        sessionStorage.setItem(key, JSON.stringify(data));
    } catch (error) {
        console.error('[Session] Failed to save session data:', error);
    }
}

/**
 * 更新会话活动时间
 */
export function updateSessionActivity(key: string): void {
    if (!isBrowser) return;

    try {
        const data = getSessionData(key);
        if (data) {
            data.last_activity = Date.now();
            saveSessionData(key, data);
        }
    } catch (error) {
        console.error('[Session] Failed to update session activity:', error);
    }
}
