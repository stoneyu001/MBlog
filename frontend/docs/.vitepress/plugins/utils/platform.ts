/**
 * 平台检测工具
 */

const isBrowser = typeof window !== 'undefined';

/**
 * 检测操作系统
 */
function detectOS(ua: string): string {
    if (/windows/i.test(ua)) return 'Windows';
    if (/macintosh|mac os x/i.test(ua)) return 'macOS';
    if (/linux/i.test(ua)) return 'Linux';
    if (/android/i.test(ua)) return 'Android';
    if (/iphone|ipad|ipod/i.test(ua)) return 'iOS';
    return 'unknown';
}

/**
 * 检测浏览器
 */
function detectBrowser(ua: string): string {
    // 按优先级排序检测
    if (/edg\/|edge\//i.test(ua)) return 'Edge';
    if (/chrome\//i.test(ua) && !/edg\/|edge\//i.test(ua)) return 'Chrome';
    if (/firefox\//i.test(ua)) return 'Firefox';
    if (/safari\//i.test(ua) && !/chrome\//i.test(ua)) return 'Safari';
    if (/opera|opr\//i.test(ua)) return 'Opera';
    return 'unknown';
}

/**
 * 检测平台信息
 * @returns 格式: "OS/Browser" (例如: "Windows/Chrome")
 */
export function detectPlatform(): string {
    if (!isBrowser) return 'server-side/unknown';

    try {
        const ua = navigator.userAgent.toLowerCase();
        const os = detectOS(ua);
        const browser = detectBrowser(ua);
        return `${os}/${browser}`;
    } catch (error) {
        return 'unknown/unknown';
    }
}
