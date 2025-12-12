/**
 * 埋点配置管理
 */

/**
 * 获取埋点 API 端点
 * 优先使用环境变量，其次使用相对路径
 */
export function getTrackingEndpoint(): string {
    // 检查是否有环境变量配置
    // @ts-ignore - Vite环境变量
    if (typeof import.meta !== 'undefined' && import.meta.env) {
        // @ts-ignore
        const envEndpoint = import.meta.env.VITE_TRACKING_ENDPOINT;
        if (envEndpoint) {
            return envEndpoint;
        }
    }

    // 默认使用相对路径
    return '/api/tracking/batch';
}

/**
 * 判断是否为开发环境
 */
export function isDevelopment(): boolean {
    // @ts-ignore - Vite环境变量
    if (typeof import.meta !== 'undefined' && import.meta.env) {
        // @ts-ignore
        return import.meta.env.DEV === true || import.meta.env.MODE === 'development';
    }
    return false;
}
