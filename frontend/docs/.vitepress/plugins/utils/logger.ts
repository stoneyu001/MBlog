/**
 * 统一日志工具
 */

export class Logger {
    private debug: boolean;
    private prefix: string;

    constructor(debug: boolean = false, prefix: string = '[Tracker]') {
        this.debug = debug;
        this.prefix = prefix;
    }

    /**
     * 普通日志
     */
    log(...args: any[]): void {
        if (this.debug && typeof console !== 'undefined') {
            console.log(`%c${this.prefix}`, 'color: #4CAF50; font-weight: bold;', ...args);
        }
    }

    /**
     * 警告日志
     */
    warn(...args: any[]): void {
        if (this.debug && typeof console !== 'undefined') {
            console.warn(`%c${this.prefix} Warning`, 'color: #FF9800; font-weight: bold;', ...args);
        }
    }

    /**
     * 错误日志（始终输出）
     */
    error(...args: any[]): void {
        if (typeof console !== 'undefined') {
            console.error(`%c${this.prefix} Error`, 'color: #F44336; font-weight: bold;', ...args);
        }
    }

    /**
     * 信息日志
     */
    info(...args: any[]): void {
        if (this.debug && typeof console !== 'undefined') {
            console.info(`%c${this.prefix} Info`, 'color: #2196F3; font-weight: bold;', ...args);
        }
    }
}
