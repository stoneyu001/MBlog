/**
 * 设备指纹生成工具
 */

const isBrowser = typeof window !== 'undefined';

/**
 * 改进的哈希函数 (FNV-1a)
 */
function hashCode(str: string): number {
    if (!str || str.length === 0) return 1;

    const FNV_PRIME = 16777619;
    const FNV_OFFSET_BASIS = 2166136261;

    let hash = FNV_OFFSET_BASIS;
    for (let i = 0; i < str.length; i++) {
        hash ^= str.charCodeAt(i);
        hash *= FNV_PRIME;
    }

    return Math.abs(hash || 1);
}

/**
 * 生成设备指纹
 * @returns 16进制字符串形式的设备指纹
 */
export function generateFingerprint(): string {
    if (!isBrowser) return 'server-side-rendering';

    try {
        // 收集设备信息
        const components = [
            navigator.userAgent || 'unknown',
            `${screen.width || 0}x${screen.height || 0}`,
            screen.colorDepth || 0,
            new Date().getTimezoneOffset(),
            navigator.language || 'unknown',
            navigator.hardwareConcurrency || 0,
            navigator.platform || 'unknown',
            navigator.cookieEnabled ? '1' : '0',
            navigator.doNotTrack || 'unknown',
            navigator.vendor || 'unknown',
            navigator.productSub || 'unknown',
            navigator.maxTouchPoints || 0,
            // @ts-ignore
            navigator.deviceMemory || 'unknown',
            // @ts-ignore
            navigator.connection?.type || 'unknown',
            // 添加随机种子
            Math.random().toString(36).substring(2) + Date.now().toString(36)
        ];

        // 生成指纹
        const rawFingerprint = components.join('|');
        const hash = hashCode(rawFingerprint);
        const fingerprint = Math.abs(hash).toString(16);

        // 验证指纹有效性
        if (!fingerprint || fingerprint.length < 8) {
            throw new Error('Invalid fingerprint generated');
        }

        return fingerprint;
    } catch (error) {
        console.error('[Fingerprint] Generation failed:', error);

        // 使用备用方案生成指纹
        const timestamp = Date.now();
        const random = Math.random();
        const fallbackComponents = [
            timestamp.toString(36),
            random.toString(36).substring(2),
            navigator.userAgent || 'unknown'
        ];

        return hashCode(fallbackComponents.join('|')).toString(16);
    }
}

/**
 * 从 localStorage 获取或创建设备指纹
 */
export function getOrCreateFingerprint(storageKey: string): string {
    if (!isBrowser) return 'server-side-rendering';

    try {
        // 尝试从 localStorage 获取
        const stored = localStorage.getItem(storageKey);
        if (stored && stored.length >= 8) {
            return stored;
        }

        // 生成新指纹
        const fingerprint = generateFingerprint();

        // 验证生成的指纹
        if (!fingerprint || fingerprint.length < 8) {
            throw new Error('Invalid fingerprint');
        }

        // 保存到 localStorage
        try {
            localStorage.setItem(storageKey, fingerprint);
        } catch (e) {
            console.error('[Fingerprint] Failed to save to localStorage:', e);
        }

        return fingerprint;
    } catch (error) {
        console.error('[Fingerprint] Error in getOrCreateFingerprint:', error);
        // 生成临时指纹
        return Date.now().toString(36) + Math.random().toString(36).substring(2);
    }
}
